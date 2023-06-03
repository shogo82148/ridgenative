package ridgenative

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"time"
)

const (
	headerAWSRequestID         = "Lambda-Runtime-Aws-Request-Id"
	headerDeadlineMS           = "Lambda-Runtime-Deadline-Ms"
	headerTraceID              = "Lambda-Runtime-Trace-Id"
	headerCognitoIdentity      = "Lambda-Runtime-Cognito-Identity"
	headerClientContext        = "Lambda-Runtime-Client-Context"
	headerInvokedFunctionARN   = "Lambda-Runtime-Invoked-Function-Arn"
	headerFunctionResponseMode = "Lambda-Runtime-Function-Response-Mode"

	trailerLambdaErrorType = "Lambda-Runtime-Function-Error-Type"
	trailerLambdaErrorBody = "Lambda-Runtime-Function-Error-Body"

	contentTypeJSON                    = "application/json"
	contentTypeBytes                   = "application/octet-stream"
	contentTypeHTTPIntegrationResponse = "application/vnd.awslambda.http-integration-response"

	apiVersion = "2018-06-01"
)

type runtimeAPIClient struct {
	baseURL    string
	userAgent  string
	httpClient *http.Client
	buffer     *bytes.Buffer
}

func newRuntimeAPIClient(address string) *runtimeAPIClient {
	client := &http.Client{
		Timeout: 0, // connections to the runtime API are never expected to time out
	}
	endpoint := "http://" + address + "/" + apiVersion + "/runtime/invocation/"
	userAgent := "aws-lambda-go/" + runtime.Version()
	return &runtimeAPIClient{
		baseURL:    endpoint,
		userAgent:  userAgent,
		httpClient: client,
		buffer:     bytes.NewBuffer(nil),
	}
}

// handlerFunc is the type of the function that handles an invoke.
type handlerFunc func(ctx context.Context, req *request) (*response, error)

func (c *runtimeAPIClient) start(ctx context.Context, h handlerFunc) error {
	for {
		invoke, err := c.next(ctx)
		if err != nil {
			return err
		}
		if err := c.handleInvoke(ctx, invoke, h); err != nil {
			return err
		}
	}
}

// next connects to the Runtime API and waits for a new invoke Request to be available.
func (c *runtimeAPIClient) next(ctx context.Context) (*invoke, error) {
	url := c.baseURL + "next"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("ridgenative: failed to construct GET request to %s: %w", url, err)
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ridgenative: failed to get the next invoke: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ridgenative: failed to GET %s: got unexpected status code: %d", url, resp.StatusCode)
	}

	c.buffer.Reset()
	_, err = c.buffer.ReadFrom(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ridgenative: failed to read the invoke payload: %w", err)
	}

	return &invoke{
		id:      resp.Header.Get(headerAWSRequestID),
		payload: c.buffer.Bytes(),
		headers: resp.Header,
	}, nil
}

// handleInvoke handles an invoke.
func (c *runtimeAPIClient) handleInvoke(ctx context.Context, invoke *invoke, h handlerFunc) error {
	// set the deadline
	deadline, err := parseDeadline(invoke)
	if err != nil {
		return c.reportFailure(ctx, invoke, lambdaErrorResponse(err))
	}
	child, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	// set the trace id
	traceID := invoke.headers.Get(headerTraceID)
	os.Setenv("_X_AMZN_TRACE_ID", traceID)
	// to keep compatibility with AWS Lambda X-Ray SDK, we need to set "x-amzn-trace-id" to the context.
	// nolint:staticcheck
	child = context.WithValue(child, "x-amzn-trace-id", traceID)

	// call the handler, marshal any returned error
	response, err := callBytesHandlerFunc(child, invoke.payload, h)
	if err != nil {
		invokeErr := lambdaErrorResponse(err)
		if err := c.reportFailure(ctx, invoke, invokeErr); err != nil {
			return err
		}
		if invokeErr.ShouldExit {
			return fmt.Errorf("calling the handler function resulted in a panic, the process should exit")
		}
		return nil
	}

	if err := c.post(ctx, invoke.id+"/response", response, contentTypeJSON); err != nil {
		return fmt.Errorf("unexpected error occurred when sending the function functionResponse to the API: %w", err)
	}

	return nil
}

func parseDeadline(invoke *invoke) (time.Time, error) {
	deadlineEpochMS, err := strconv.ParseInt(invoke.headers.Get(headerDeadlineMS), 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("ridgenative: failed to parse deadline: %w", err)
	}
	return time.UnixMilli(deadlineEpochMS), nil
}

// post posts body to the Runtime API at the given path.
func (c *runtimeAPIClient) post(ctx context.Context, path string, body []byte, contentType string) error {
	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("ridgenative: failed to construct POST request to %s: %w", url, err)
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Content-Type", contentType)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ridgenative: failed to POST to %s: %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("ridgenative: failed to POST to %s: got unexpected status code: %d", url, resp.StatusCode)
	}

	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		return fmt.Errorf("ridgenative: something went wrong reading the POST response from %s: %w", url, err)
	}

	return nil
}

// reportFailure reports the error to the Runtime API.
func (c *runtimeAPIClient) reportFailure(ctx context.Context, invoke *invoke, invokeErr *invokeResponseError) error {
	body, err := json.Marshal(invokeErr)
	if err != nil {
		return fmt.Errorf("ridgenative: failed to marshal the function error: %w", err)
	}
	log.Printf("%s", body)
	if err := c.post(ctx, invoke.id+"/error", body, contentTypeJSON); err != nil {
		return fmt.Errorf("ridgenative: unexpected error occurred when sending the function error to the API: %w", err)
	}
	return nil
}

type handlerFuncSteaming func(ctx context.Context, req *request, w *io.PipeWriter) error

func (c *runtimeAPIClient) startStreaming(ctx context.Context, h handlerFuncSteaming) error {
	for {
		invoke, err := c.next(ctx)
		if err != nil {
			return err
		}
		if err := c.handleInvokeStreaming(ctx, invoke, h); err != nil {
			return err
		}
	}
}

// handleInvoke handles an invoke.
func (c *runtimeAPIClient) handleInvokeStreaming(ctx context.Context, invoke *invoke, h handlerFuncSteaming) error {
	// set the deadline
	deadline, err := parseDeadline(invoke)
	if err != nil {
		return c.reportFailure(ctx, invoke, lambdaErrorResponse(err))
	}
	child, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	// set the trace id
	traceID := invoke.headers.Get(headerTraceID)
	os.Setenv("_X_AMZN_TRACE_ID", traceID)
	// to keep compatibility with AWS Lambda X-Ray SDK, we need to set "x-amzn-trace-id" to the context.
	// nolint:staticcheck
	child = context.WithValue(child, "x-amzn-trace-id", traceID)

	// call the handler, marshal any returned error
	response, err := callHandlerFuncSteaming(child, invoke.payload, h)
	if err != nil {
		invokeErr := lambdaErrorResponse(err)
		if err := c.reportFailure(ctx, invoke, invokeErr); err != nil {
			return err
		}
		if invokeErr.ShouldExit {
			return fmt.Errorf("calling the handler function resulted in a panic, the process should exit")
		}
		return nil
	}

	if err := c.postStreaming(ctx, invoke.id+"/response", response, contentTypeHTTPIntegrationResponse); err != nil {
		return fmt.Errorf("unexpected error occurred when sending the function functionResponse to the API: %w", err)
	}

	return nil
}

// postStreaming posts body to the Runtime API at the given path.
func (c *runtimeAPIClient) postStreaming(ctx context.Context, path string, body io.ReadCloser, contentType string) error {
	b := newErrorCapturingReader(body)
	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, b)
	if err != nil {
		return fmt.Errorf("ridgenative: failed to construct POST request to %s: %w", url, err)
	}
	req.Trailer = b.trailer
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set(headerFunctionResponseMode, "streaming")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ridgenative: failed to POST to %s: %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("ridgenative: failed to POST to %s: got unexpected status code: %d", url, resp.StatusCode)
	}

	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		return fmt.Errorf("ridgenative: something went wrong reading the POST response from %s: %w", url, err)
	}

	return nil
}

// errorCapturingReader is a reader that captures the first error returned by the underlying reader.
type errorCapturingReader struct {
	reader  io.ReadCloser
	err     error
	trailer http.Header
}

func newErrorCapturingReader(r io.ReadCloser) *errorCapturingReader {
	return &errorCapturingReader{
		reader:  r,
		trailer: http.Header{},
	}
}

func (r *errorCapturingReader) Read(p []byte) (int, error) {
	if r.reader == nil {
		return 0, io.EOF
	}
	if r.err != nil {
		return 0, r.err
	}

	n, err := r.reader.Read(p)
	if err != nil && errors.Is(err, io.EOF) {
		// capture the error
		r.err = err
		lambdaErr := lambdaErrorResponse(err)
		body, err := json.Marshal(lambdaErr)
		if err != nil {
			// marshaling lambdaErr always succeeds
			// because lambdaErr doesn't have any functions and channels.
			panic(err)
		}
		r.trailer.Set(trailerLambdaErrorType, lambdaErr.Type)
		r.trailer.Set(trailerLambdaErrorBody, base64.StdEncoding.EncodeToString(body))
	}
	return n, err
}

func (r *errorCapturingReader) Close() error {
	return r.reader.Close()
}
