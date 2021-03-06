package ridgenative

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type lambdaFunction struct {
	mux http.Handler
	// buffer for string data
	builder strings.Builder

	// buffer for binary data
	buffer bytes.Buffer
	out    []byte
}

type request struct {
	// Common Fields
	HTTPMethod                      string              `json:"httpMethod"`
	Path                            string              `json:"path"`
	QueryStringParameters           map[string]string   `json:"queryStringParameters,omitempty"`
	MultiValueQueryStringParameters map[string][]string `json:"multiValueQueryStringParameters,omitempty"`
	Headers                         map[string]string   `json:"headers,omitempty"`
	MultiValueHeaders               map[string][]string `json:"multiValueHeaders,omitempty"`
	IsBase64Encoded                 bool                `json:"isBase64Encoded"`
	Body                            string              `json:"body"`
	RequestContext                  requestContext      `json:"requestContext"`

	// for API Gateway events
	Resource       string            `json:"resource"`
	PathParameters map[string]string `json:"pathParameters"`
	StageVariables map[string]string `json:"stageVariables"`

	// for API Gateway v2 events
	Version        string `json:"version"`
	RawPath        string `json:"rawPath"`
	RawQueryString string `json:"rawQueryString"`
}

type requestContext struct {
	// for ALB Target Group events
	ELB *events.ELBContext `json:"elb"`

	// for API Gateway events
	AccountID    string                           `json:"accountId"`
	ResourceID   string                           `json:"resourceId"`
	Stage        string                           `json:"stage"`
	RequestID    string                           `json:"requestId"`
	Identity     events.APIGatewayRequestIdentity `json:"identity"`
	ResourcePath string                           `json:"resourcePath"`
	Authorizer   map[string]interface{}           `json:"authorizer"`
	HTTPMethod   string                           `json:"httpMethod"`
	APIID        string                           `json:"apiId"` // The API Gateway rest API Id

	// for API Gateway v2 events
	HTTP *requestContextHTTP `json:"http"`
}

type requestContextHTTP struct {
	Method    string `json:"method"`
	Path      string `json:"path"`
	Protocol  string `json:"protocol"`
	SourceIP  string `json:"sourceIp"`
	UserAgent string `json:"userAgent"`
}

func (f *lambdaFunction) httpRequest(ctx context.Context, r request) (*http.Request, error) {
	if r.RequestContext.HTTP != nil {
		// API Gateway v2
		return f.httpRequestAPIGatewayV2(ctx, r)
	}
	// API Gateway v1 or ALB

	// decode header
	var headers http.Header
	if len(r.MultiValueHeaders) > 0 {
		headers = make(http.Header, len(r.MultiValueHeaders))
		for k, v := range r.MultiValueHeaders {
			headers[textproto.CanonicalMIMEHeaderKey(k)] = v
		}
	} else {
		// fall back to headers
		headers = make(http.Header, len(r.Headers))
		for k, v := range r.Headers {
			headers[textproto.CanonicalMIMEHeaderKey(k)] = []string{v}
		}
	}

	// decode query string
	var values url.Values
	if len(r.MultiValueQueryStringParameters) > 0 {
		values = make(url.Values, len(r.MultiValueQueryStringParameters))
		for k, v := range r.MultiValueQueryStringParameters {
			values[k] = v
		}
	} else if len(r.QueryStringParameters) > 0 {
		// fall back to queryStringParameters
		values = make(url.Values, len(r.QueryStringParameters))
		for k, v := range r.QueryStringParameters {
			values[k] = []string{v}
		}
	}

	// build uri
	uri := r.Path
	if len(values) > 0 {
		uri = uri + "?" + values.Encode()
	}
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	// build body
	var contentLength int64
	var body io.ReadCloser
	if r.Body != "" {
		var reader io.Reader
		if r.IsBase64Encoded {
			f.buffer.Reset()
			f.buffer.WriteString(r.Body)
			n := base64.StdEncoding.DecodedLen(len(r.Body))
			out := f.out
			if cap(out) < n {
				out = make([]byte, n)
			} else {
				out = out[:n]
			}
			n, err := base64.StdEncoding.Decode(out, f.buffer.Bytes())
			f.out = out
			if err != nil {
				return nil, err
			}
			contentLength = int64(n)
			reader = bytes.NewReader(out[:n])
		} else {
			contentLength = int64(len(r.Body))
			reader = io.Reader(strings.NewReader(r.Body))
		}
		body = ioutil.NopCloser(reader)
	} else {
		body = http.NoBody
	}

	req := &http.Request{
		Method:        r.HTTPMethod,
		Proto:         "HTTP/1.0",
		ProtoMajor:    1,
		ProtoMinor:    0,
		Header:        headers,
		RemoteAddr:    r.RequestContext.Identity.SourceIP,
		ContentLength: contentLength,
		Body:          body,
		RequestURI:    uri,
		URL:           u,
		Host:          headers.Get("Host"),
	}
	req = req.WithContext(ctx)
	return req, nil
}

func (f *lambdaFunction) httpRequestAPIGatewayV2(ctx context.Context, r request) (*http.Request, error) {
	headers := make(http.Header, len(r.Headers))
	for k, v := range r.Headers {
		headers[textproto.CanonicalMIMEHeaderKey(k)] = []string{v}
	}

	// build uri
	uri := r.RequestContext.HTTP.Path
	rawURI := r.RawPath
	if r.RawQueryString != "" {
		uri = uri + "?" + r.RawQueryString
		rawURI = rawURI + "?" + r.RawQueryString
	}
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	// build body
	var contentLength int64
	var body io.ReadCloser
	if r.Body != "" {
		var reader io.Reader
		if r.IsBase64Encoded {
			f.buffer.Reset()
			f.buffer.WriteString(r.Body)
			n := base64.StdEncoding.DecodedLen(len(r.Body))
			out := f.out
			if cap(out) < n {
				out = make([]byte, n)
			} else {
				out = out[:n]
			}
			n, err := base64.StdEncoding.Decode(out, f.buffer.Bytes())
			f.out = out
			if err != nil {
				return nil, err
			}
			contentLength = int64(n)
			reader = bytes.NewReader(out[:n])
		} else {
			contentLength = int64(len(r.Body))
			reader = io.Reader(strings.NewReader(r.Body))
		}
		body = ioutil.NopCloser(reader)
	} else {
		body = http.NoBody
	}

	req := &http.Request{
		Method:        r.RequestContext.HTTP.Method,
		Proto:         "HTTP/1.0",
		ProtoMajor:    1,
		ProtoMinor:    0,
		Header:        headers,
		RemoteAddr:    r.RequestContext.HTTP.SourceIP,
		ContentLength: contentLength,
		Body:          body,
		RequestURI:    rawURI,
		URL:           u,
		Host:          headers.Get("Host"),
	}
	req = req.WithContext(ctx)
	return req, nil
}

type responseWriter struct {
	w           io.Writer
	isBinary    bool
	wroteHeader bool
	header      http.Header
	statusCode  int
	lambda      *lambdaFunction
}

type response struct {
	StatusCode        int                 `json:"statusCode"`
	Headers           map[string]string   `json:"headers"`
	MultiValueHeaders map[string][]string `json:"multiValueHeaders"`
	Body              string              `json:"body"`
	IsBase64Encoded   bool                `json:"isBase64Encoded"`
}

func (f *lambdaFunction) newResponseWriter() *responseWriter {
	f.builder.Reset()
	f.buffer.Reset()
	return &responseWriter{
		header: make(http.Header, 1),
		lambda: f,
	}
}

// relevantCaller searches the call stack for the first function outside of net/http.
// The purpose of this function is to provide more helpful error messages.
func relevantCaller() runtime.Frame {
	pc := make([]uintptr, 16)
	n := runtime.Callers(1, pc)
	frames := runtime.CallersFrames(pc[:n])
	var frame runtime.Frame
	for {
		frame, more := frames.Next()
		if !strings.HasPrefix(frame.Function, "net/http.") {
			return frame
		}
		if !more {
			break
		}
	}
	return frame
}

func (rw *responseWriter) Header() http.Header {
	return rw.header
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.wroteHeader {
		caller := relevantCaller()
		log.Printf("ridgenative: superfluous response.WriteHeader call from %s (%s:%d)", caller.Function, path.Base(caller.File), caller.Line)
		return
	}
	rw.statusCode = code
	rw.wroteHeader = true
	if typ := rw.header.Get("Content-Type"); typ != "" {
		rw.isBinary = isBinary(typ)
		if rw.isBinary {
			rw.w = &rw.lambda.buffer
		} else {
			rw.w = &rw.lambda.builder
		}
	}
}

func (rw *responseWriter) Write(data []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	if rw.w != nil {
		return rw.w.Write(data)
	}

	f := rw.lambda
	rest := 512 - f.buffer.Len()
	if len(data) < rest {
		return f.buffer.Write(data)
	}
	n1, _ := f.buffer.Write(data[:rest])
	rw.detectContentType()
	n2, _ := rw.w.Write(data[rest:])
	return n1 + n2, nil
}

func (rw *responseWriter) lambdaResponse() (response, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	if rw.w == nil {
		rw.detectContentType()
	}

	var body string
	if rw.isBinary {
		out := rw.lambda.out
		l := base64.StdEncoding.EncodedLen(rw.lambda.buffer.Len())
		if cap(out) < l {
			out = make([]byte, l)
		} else {
			out = out[:l]
		}
		base64.StdEncoding.Encode(out, rw.lambda.buffer.Bytes())
		body = string(out)
		rw.lambda.out = out
	} else {
		body = rw.lambda.builder.String()
	}

	// fall back to headers if multiValueHeaders is not available
	h := make(map[string]string, len(rw.header))
	for key := range rw.header {
		h[key] = rw.header.Get(key)
	}

	return response{
		StatusCode:        rw.statusCode,
		Headers:           h,
		MultiValueHeaders: map[string][]string(rw.header),
		Body:              body,
		IsBase64Encoded:   rw.isBinary,
	}, nil
}

func (rw *responseWriter) detectContentType() {
	contentType := http.DetectContentType(rw.lambda.buffer.Bytes())
	rw.header.Set("Content-Type", contentType)
	rw.isBinary = isBinary(contentType)
	if rw.isBinary {
		rw.w = &rw.lambda.buffer
	} else {
		rw.w = &rw.lambda.builder
		rw.lambda.buffer.WriteTo(rw.w)
	}
}

// assume text/*, application/json, application/javascript, application/xml, */*+json, */*+xml as text
func isBinary(contentType string) bool {
	i := strings.Index(contentType, ";")
	if i == -1 {
		i = len(contentType)
	}
	mediaType := strings.TrimSpace(contentType[:i])
	i = strings.Index(mediaType, "/")
	if i == -1 {
		i = len(mediaType)
	}
	mainType := mediaType[:i]

	if strings.EqualFold(mainType, "text") {
		return false
	}
	if strings.EqualFold(mediaType, "application/json") {
		return false
	}
	if strings.EqualFold(mediaType, "application/javascript") {
		return false
	}
	if strings.EqualFold(mediaType, "application/xml") {
		return false
	}

	i = strings.LastIndex(mediaType, "+")
	if i == -1 {
		i = 0
	}
	suffix := mediaType[i:]
	if strings.EqualFold(suffix, "+json") {
		return false
	}
	if strings.EqualFold(suffix, "+xml") {
		return false
	}
	return true
}

func (f *lambdaFunction) lambdaHandler(ctx context.Context, req request) (response, error) {
	r, err := f.httpRequest(ctx, req)
	if err != nil {
		return response{}, err
	}
	rw := f.newResponseWriter()
	f.mux.ServeHTTP(rw, r)
	return rw.lambdaResponse()
}

func newLambdaFunction(mux http.Handler) *lambdaFunction {
	return &lambdaFunction{
		mux: mux,
	}
}

// ListenAndServe starts HTTP server.
//
// If AWS_EXECUTION_ENV environment value is defined, it wait for new AWS Lambda events and handle it as HTTP requests.
// The format of the events is compatible with Amazon API Gateway Lambda proxy integration and Application Load Balancers.
// See AWS documents for details.
//
// https://docs.aws.amazon.com/apigateway/latest/developerguide/set-up-lambda-proxy-integrations.html
//
// https://docs.aws.amazon.com/elasticloadbalancing/latest/application/lambda-functions.html
//
// If AWS_EXECUTION_ENV environment value is NOT defined, it just calls http.ListenAndServe.
//
// The handler is typically nil, in which case the DefaultServeMux is used.
func ListenAndServe(address string, mux http.Handler) error {
	go1 := os.Getenv("AWS_EXECUTION_ENV")      // run on go1.x runtime
	al2 := os.Getenv("AWS_LAMBDA_RUNTIME_API") // run on provided or provided.al2 runtime
	if go1 == "" && al2 == "" {
		// fall back to normal HTTP server.
		return http.ListenAndServe(address, mux)
	}
	if mux == nil {
		mux = http.DefaultServeMux
	}
	f := newLambdaFunction(mux)
	lambda.Start(f.lambdaHandler)
	panic("do not reach")
}
