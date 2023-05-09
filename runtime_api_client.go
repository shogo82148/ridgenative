package ridgenative

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"runtime"
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
		client:  c,
	}, nil
}

func (c *runtimeAPIClient) post(url string, body io.Reader, contentType string) error {
	b := body
	req, err := http.NewRequest(http.MethodPost, url, b)
	if err != nil {
		return fmt.Errorf("failed to construct POST request to %s: %v", url, err)
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Content-Type", contentType)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to POST to %s: %v", url, err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("runtime API client failed to close %s response body: %v", url, err)
		}
	}()
	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("failed to POST to %s: got unexpected status code: %d", url, resp.StatusCode)
	}

	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		return fmt.Errorf("something went wrong reading the POST response from %s: %v", url, err)
	}

	return nil
}
