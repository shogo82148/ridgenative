package ridgenative

import (
	"bytes"
	"context"
	"encoding/json"
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
	headerAWSRequestID       = "Lambda-Runtime-Aws-Request-Id"
	headerDeadlineMS         = "Lambda-Runtime-Deadline-Ms"
	headerTraceID            = "Lambda-Runtime-Trace-Id"
	headerCognitoIdentity    = "Lambda-Runtime-Cognito-Identity"
	headerClientContext      = "Lambda-Runtime-Client-Context"
	headerInvokedFunctionARN = "Lambda-Runtime-Invoked-Function-Arn"
	trailerLambdaErrorType   = "Lambda-Runtime-Function-Error-Type"
	trailerLambdaErrorBody   = "Lambda-Runtime-Function-Error-Body"
	contentTypeJSON          = "application/json"
	contentTypeBytes         = "application/octet-stream"
	apiVersion               = "2018-06-01"
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

func (c *runtimeAPIClient) start(h handlerFunc) error {
	for {
		invoke, err := c.next()
		if err != nil {
			return err
		}
		if err := c.handleInvoke(invoke, h); err != nil {
			return err
		}
	}
}

// next connects to the Runtime API and waits for a new invoke Request to be available.
func (c *runtimeAPIClient) next() (*invoke, error) {
	url := c.baseURL + "next"
	req, err := http.NewRequest(http.MethodGet, url, nil)
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

// handleInvoke returns an error if the function panics, or some other non-recoverable error occurred
func (c *runtimeAPIClient) handleInvoke(invoke *invoke, h handlerFunc) error {
	// set the deadline
	deadline, err := parseDeadline(invoke)
	if err != nil {
		return c.reportFailure(invoke, lambdaErrorResponse(err))
	}
	ctx, cancel := context.WithDeadline(context.TODO(), deadline)
	defer cancel()

	// set the trace id
	traceID := invoke.headers.Get(headerTraceID)
	os.Setenv("_X_AMZN_TRACE_ID", traceID)
	// to keep compatibility with AWS Lambda X-Ray SDK, we need to set "x-amzn-trace-id" to the context.
	// nolint:staticcheck
	ctx = context.WithValue(ctx, "x-amzn-trace-id", traceID)

	// call the handler, marshal any returned error
	response, err := callBytesHandlerFunc(ctx, invoke.payload, h)
	if err != nil {
		invokeErr := lambdaErrorResponse(err)
		if err := c.reportFailure(invoke, invokeErr); err != nil {
			return err
		}
		if invokeErr.ShouldExit {
			return fmt.Errorf("calling the handler function resulted in a panic, the process should exit")
		}
		return nil
	}

	if err := c.post(invoke.id+"/response", response, contentTypeJSON); err != nil {
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
func (c *runtimeAPIClient) post(path string, body []byte, contentType string) error {
	url := c.baseURL + path
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("ridgenative: failed to construct POST request to %s: %v", url, err)
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Content-Type", contentType)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ridgenative: failed to POST to %s: %v", url, err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("ridgenative: runtime API client failed to close %s response body: %v", url, err)
		}
	}()
	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("ridgenative: failed to POST to %s: got unexpected status code: %d", url, resp.StatusCode)
	}

	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		return fmt.Errorf("ridgenative: something went wrong reading the POST response from %s: %v", url, err)
	}

	return nil
}

func (c *runtimeAPIClient) reportFailure(invoke *invoke, invokeErr *invokeResponseError) error {
	body, err := json.Marshal(invokeErr)
	if err != nil {
		return fmt.Errorf("ridgenative: failed to marshal the function error: %w", err)
	}
	log.Printf("%s", body)
	if err := c.post(invoke.id+"/error", body, contentTypeJSON); err != nil {
		return fmt.Errorf("ridgenative: unexpected error occurred when sending the function error to the API: %w", err)
	}
	return nil
}
