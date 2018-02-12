package ridgenative

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
)

type request struct {
	Body                  *string           `json:"body"`
	Headers               map[string]string `json:"headers"`
	HTTPMethod            string            `json:"httpMethod"`
	IsBase64Encoded       bool              `json:"isBase64Encoded"`
	Path                  string            `json:"path"`
	PathParameters        map[string]string `json:"pathParameters"`
	QueryStringParameters map[string]string `json:"queryStringParameters"`
	RequestContext        requestContext    `json:"requestContext"`
	Resource              string            `json:"resource"`
	StageVariables        map[string]string `json:"stageVariables"`
}

type requestContext struct {
	AccountID        string            `json:"accountId"`
	APIID            string            `json:"apiId"`
	HTTPMethod       string            `json:"httpMethod"`
	Identity         map[string]string `json:"identity"`
	Path             string            `json:"path"`
	Protocol         string            `json:"protocol"`
	RequestID        string            `json:"requestId"`
	RequestTime      string            `json:"requestTime"`
	RequestTimeEpoch int64             `json:"requestTimeEpoch"`
	ResourceID       string            `json:"resourceId"`
	ResourcePath     string            `json:"resourcePath"`
	Stage            string            `json:"stage"`
}

type response struct {
	IsBase64Encoded bool              `json:"isBase64Encoded"`
	StatusCode      int               `json:"statusCode"`
	Headers         map[string]string `json:"headers"`
	Body            string            `json:"body"`
}

type lambdaFunction struct {
	prefix string
	mux    http.Handler
}

func (r *request) httpRequest() (*http.Request, error) {
	headers := http.Header{}
	for k, v := range r.Headers {
		headers.Add(k, v)
	}

	values := url.Values{}
	for k, v := range r.QueryStringParameters {
		values.Add(k, v)
	}
	uri := r.Path
	if len(values) > 0 {
		uri = uri + "?" + values.Encode()
	}
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	var contentLength int64
	var body io.ReadCloser
	if r.Body != nil {
		contentLength = int64(len(*r.Body))
		reader := io.Reader(strings.NewReader(*r.Body))
		if r.IsBase64Encoded {
			// ignore padding
			if contentLength > 0 && (*r.Body)[contentLength-1] == '=' {
				contentLength--
			}
			if contentLength > 0 && (*r.Body)[contentLength-1] == '=' {
				contentLength--
			}
			if contentLength > 0 && (*r.Body)[contentLength-1] == '=' {
				contentLength--
			}

			contentLength = contentLength * 3 / 4
			reader = base64.NewDecoder(base64.StdEncoding, reader)
		}
		body = ioutil.NopCloser(reader)
	} else {
		body = http.NoBody
	}

	return &http.Request{
		Method:        r.HTTPMethod,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        headers,
		RemoteAddr:    r.RequestContext.Identity["sourceIp"],
		ContentLength: contentLength,
		Body:          body,
		RequestURI:    uri,
		URL:           u,
	}, nil
}

type responseWriter struct {
	*bytes.Buffer
	header     http.Header
	statusCode int
}

func newResponseWriter() *responseWriter {
	return &responseWriter{
		Buffer:     &bytes.Buffer{},
		header:     http.Header{},
		statusCode: http.StatusOK,
	}
}

func (rw *responseWriter) Header() http.Header {
	return rw.header
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
}

func (rw *responseWriter) lambdaResponse() (*response, error) {
	h := make(map[string]string, len(rw.header))
	for key := range rw.header {
		h[key] = rw.header.Get(key)
	}

	return &response{
		StatusCode: rw.statusCode,
		Headers:    h,
		Body:       rw.String(),
	}, nil
}

func (f lambdaFunction) lambdaHandler(ctx context.Context, req *request) (*response, error) {
	r, err := req.httpRequest()
	if err != nil {
		return nil, err
	}
	r = r.WithContext(ctx)
	rw := newResponseWriter()
	f.mux.ServeHTTP(rw, r)
	return rw.lambdaResponse()
}

// Run runs http handler on Apex or net/http's server.
func Run(address, prefix string, mux http.Handler) {
	if mux == nil {
		mux = http.DefaultServeMux
	}
	f := lambdaFunction{prefix: prefix, mux: mux}
	lambda.Start(f.lambdaHandler)
}
