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

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type lambdaFunction struct {
	prefix string
	mux    http.Handler
}

func httpRequest(r events.APIGatewayProxyRequest) (*http.Request, error) {
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
	if r.Body != "" {
		contentLength = int64(len(r.Body))
		reader := io.Reader(strings.NewReader(r.Body))
		if r.IsBase64Encoded {
			// ignore padding
			if contentLength > 0 && r.Body[contentLength-1] == '=' {
				contentLength--
			}
			if contentLength > 0 && r.Body[contentLength-1] == '=' {
				contentLength--
			}
			if contentLength > 0 && r.Body[contentLength-1] == '=' {
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
		RemoteAddr:    r.RequestContext.Identity.SourceIP,
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

func (rw *responseWriter) lambdaResponse() (events.APIGatewayProxyResponse, error) {
	h := make(map[string]string, len(rw.header))
	for key := range rw.header {
		h[key] = rw.header.Get(key)
	}

	return events.APIGatewayProxyResponse{
		StatusCode: rw.statusCode,
		Headers:    h,
		Body:       rw.String(),
	}, nil
}

func (f lambdaFunction) lambdaHandler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	r, err := httpRequest(req)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
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
