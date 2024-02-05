package ridgenative

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path"
	"runtime"
	"strings"
)

type lambdaFunction struct {
	mux http.Handler
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
	Version        string   `json:"version"`
	RawPath        string   `json:"rawPath"`
	RawQueryString string   `json:"rawQueryString"`
	Cookies        []string `json:"cookies"`
}

type requestContext struct {
	// for API Gateway events
	AccountID    string                 `json:"accountId"`
	ResourceID   string                 `json:"resourceId"`
	Stage        string                 `json:"stage"`
	RequestID    string                 `json:"requestId"`
	Identity     requestIdentity        `json:"identity"`
	ResourcePath string                 `json:"resourcePath"`
	Authorizer   map[string]interface{} `json:"authorizer"`
	HTTPMethod   string                 `json:"httpMethod"`
	APIID        string                 `json:"apiId"` // The API Gateway rest API Id

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

// apiIGatewayRequestIdentity contains identity information for the request caller.
type requestIdentity struct {
	CognitoIdentityPoolID         string `json:"cognitoIdentityPoolId"`
	AccountID                     string `json:"accountId"`
	CognitoIdentityID             string `json:"cognitoIdentityId"`
	Caller                        string `json:"caller"`
	APIKey                        string `json:"apiKey"`
	APIKeyID                      string `json:"apiKeyId"`
	AccessKey                     string `json:"accessKey"`
	SourceIP                      string `json:"sourceIp"`
	CognitoAuthenticationType     string `json:"cognitoAuthenticationType"`
	CognitoAuthenticationProvider string `json:"cognitoAuthenticationProvider"`
	UserArn                       string `json:"userArn"` //nolint: stylecheck
	UserAgent                     string `json:"userAgent"`
	User                          string `json:"user"`
}

func isV2Request(r *request) bool {
	return r.Version == "2" || strings.HasPrefix(r.Version, "2.")
}

func (f *lambdaFunction) httpRequestV1(ctx context.Context, r *request) (*http.Request, error) {
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
	body, contentLength, err := f.decodeBody(r)
	if err != nil {
		return nil, err
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

func (f *lambdaFunction) httpRequestV2(ctx context.Context, r *request) (*http.Request, error) {
	// build headers
	headers := make(http.Header, len(r.Headers))
	for k, v := range r.Headers {
		headers.Set(k, v)
	}

	// build cookies
	if len(r.Cookies) > 0 {
		headers.Set("Cookie", strings.Join(r.Cookies, ";"))
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
	body, contentLength, err := f.decodeBody(r)
	if err != nil {
		return nil, err
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

func (f *lambdaFunction) decodeBody(r *request) (body io.ReadCloser, contentLength int64, err error) {
	if r.Body == "" {
		body = http.NoBody
		return
	}

	var reader io.Reader
	if r.IsBase64Encoded {
		var b []byte
		b, err = base64.StdEncoding.DecodeString(r.Body)
		if err != nil {
			return
		}
		contentLength = int64(len(b))
		reader = bytes.NewReader(b)
	} else {
		contentLength = int64(len(r.Body))
		reader = strings.NewReader(r.Body)
	}
	body = io.NopCloser(reader)
	return
}

type responseWriter struct {
	w           bytes.Buffer
	isBinary    bool
	wroteHeader bool
	header      http.Header
	statusCode  int
}

type response struct {
	StatusCode        int                 `json:"statusCode,omitempty"`
	Headers           map[string]string   `json:"headers,omitempty"`
	MultiValueHeaders map[string][]string `json:"multiValueHeaders,omitempty"`
	Body              string              `json:"body,omitempty"`
	IsBase64Encoded   bool                `json:"isBase64Encoded,omitempty"`
	Cookies           []string            `json:"cookies,omitempty"`
}

func newResponseWriter() *responseWriter {
	return &responseWriter{
		header: make(http.Header, 1),
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
}

func (rw *responseWriter) Write(data []byte) (int, error) {
	return rw.w.Write(data)
}

func (rw *responseWriter) lambdaResponseV1() (*response, error) {
	body := rw.encodeBody()

	// fall back to headers if multiValueHeaders is not available
	h := make(map[string]string, len(rw.header))
	for key, value := range rw.header {
		if key == "Set-Cookie" {
			// we can't fold Set-Cookie header, because the %x2C (",") character is used
			// by Set-Cookie in a way that conflicts with such folding.
			if len(value) > 0 {
				h[key] = value[0]
			}
			continue
		}
		h[key] = strings.Join(value, ", ")
	}

	return &response{
		StatusCode:        rw.statusCode,
		Headers:           h,
		MultiValueHeaders: map[string][]string(rw.header),
		Body:              body,
		IsBase64Encoded:   rw.isBinary,
	}, nil
}

func (rw *responseWriter) lambdaResponseV2() (*response, error) {
	body := rw.encodeBody()

	// multiValueHeaders is not available in V2; fall back to headers
	h := make(map[string]string, len(rw.header))
	for key, value := range rw.header {
		if key == "Set-Cookie" {
			continue
		}
		h[key] = strings.Join(value, ", ")
	}

	cookies := rw.header.Values("Set-Cookie")

	return &response{
		StatusCode:      rw.statusCode,
		Headers:         h,
		Cookies:         cookies,
		Body:            body,
		IsBase64Encoded: rw.isBinary,
	}, nil
}

func (rw *responseWriter) encodeBody() string {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}

	if typ := rw.header.Get("Content-Type"); typ != "" {
		rw.isBinary = isBinary(typ)
	} else {
		rw.detectContentType()
	}

	if rw.isBinary {
		return base64.StdEncoding.EncodeToString(rw.w.Bytes())
	} else {
		return rw.w.String()
	}
}

func (rw *responseWriter) detectContentType() {
	contentType := http.DetectContentType(rw.w.Bytes())
	rw.header.Set("Content-Type", contentType)
	rw.isBinary = isBinary(contentType)
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
	if strings.EqualFold(mediaType, "application/yaml") {
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
	if strings.EqualFold(suffix, "+yaml") {
		return false
	}
	if strings.EqualFold(suffix, "+xml") {
		return false
	}
	return true
}

func (f *lambdaFunction) lambdaHandler(ctx context.Context, req *request) (*response, error) {
	if isV2Request(req) {
		// Lambda Function URLs or API Gateway v2
		r, err := f.httpRequestV2(ctx, req)
		if err != nil {
			return nil, err
		}
		rw := newResponseWriter()
		f.mux.ServeHTTP(rw, r)
		return rw.lambdaResponseV2()
	} else {
		// API Gateway v1 or ALB
		r, err := f.httpRequestV1(ctx, req)
		if err != nil {
			return nil, err
		}
		rw := newResponseWriter()
		f.mux.ServeHTTP(rw, r)
		return rw.lambdaResponseV1()
	}
}

type streamingResponse struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers,omitempty"`
	Cookies    []string          `json:"cookies,omitempty"`
}

// streamingResponseWriter is a http.ResponseWriter that supports streaming.
type streamingResponseWriter struct {
	w           *io.PipeWriter
	buf         *bufio.Writer
	wroteHeader bool
	header      http.Header
	statusCode  int
	err         error

	// prelude is the first part of the body.
	// it is used for detecting content-type.
	prelude []byte
}

func newStreamingResponseWriter(w *io.PipeWriter) *streamingResponseWriter {
	return &streamingResponseWriter{
		w:       w,
		buf:     bufio.NewWriter(w),
		header:  make(http.Header, 1),
		prelude: make([]byte, 0, 512),
	}
}

func (rw *streamingResponseWriter) Header() http.Header {
	return rw.header
}

func (rw *streamingResponseWriter) WriteHeader(code int) {
	if rw.wroteHeader {
		caller := relevantCaller()
		log.Printf("ridgenative: superfluous response.WriteHeader call from %s (%s:%d)", caller.Function, path.Base(caller.File), caller.Line)
		return
	}
	if rw.err != nil {
		return
	}

	if !rw.hasContentType() {
		rw.header.Set("Content-Type", http.DetectContentType(rw.prelude))
	}

	rw.wroteHeader = true
	rw.statusCode = code

	// build the prelude
	h := make(map[string]string, len(rw.header))
	for key, value := range rw.header {
		if key == "Set-Cookie" {
			continue
		}
		h[key] = strings.Join(value, ", ")
	}
	cookies := rw.header.Values("Set-Cookie")
	r := &streamingResponse{
		StatusCode: code,
		Headers:    h,
		Cookies:    cookies,
	}

	data, err := json.Marshal(r)
	if err != nil {
		rw.err = fmt.Errorf("ridgenative: failed to marshal response: %w", err)
		return
	}
	if _, err := rw.buf.Write(data); err != nil {
		rw.err = err
		return
	}
	if _, err := rw.buf.WriteString("\x00\x00\x00\x00\x00\x00\x00\x00"); err != nil {
		rw.err = err
		return
	}
	if len(rw.prelude) != 0 {
		if _, err := rw.buf.Write(rw.prelude); err != nil {
			rw.err = err
			return
		}
	}
	if err := rw.buf.Flush(); err != nil {
		rw.err = err
	}
}

func (rw *streamingResponseWriter) hasContentType() bool {
	return rw.header.Get("Content-Type") != ""
}

func (rw *streamingResponseWriter) Write(data []byte) (int, error) {
	var m int
	if !rw.wroteHeader {
		if rw.hasContentType() {
			rw.WriteHeader(http.StatusOK)
		} else {
			// save the first part of the body for detecting content-type.
			data0 := data
			if len(rw.prelude)+len(data0) > cap(rw.prelude) {
				data0 = data0[:cap(rw.prelude)-len(rw.prelude)]
			}
			rw.prelude = append(rw.prelude, data0...)

			if len(rw.prelude) == cap(rw.prelude) {
				rw.WriteHeader(http.StatusOK)
			}
			m = len(data0)
			data = data[m:]
			if len(data) == 0 {
				return m, nil
			}
		}
	}
	n, err := rw.buf.Write(data)
	return n + m, err
}

func (rw *streamingResponseWriter) closeWithError(err error) error {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	if rw.err != nil {
		err = rw.err
	}
	if err0 := rw.buf.Flush(); err0 != nil {
		err = err0
	}
	return rw.w.CloseWithError(err)
}

func (rw *streamingResponseWriter) close() error {
	return rw.closeWithError(nil)
}

func (rw *streamingResponseWriter) Flush() {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	rw.buf.Flush()
}

func (f *lambdaFunction) lambdaHandlerStreaming(ctx context.Context, req *request, w *io.PipeWriter) (string, error) {
	r, err := f.httpRequestV2(ctx, req)
	if err != nil {
		return "", err
	}
	go func() {
		rw := newStreamingResponseWriter(w)
		defer func() {
			if v := recover(); v != nil {
				_ = rw.closeWithError(lambdaPanicResponse(v))
			} else {
				_ = rw.close()
			}
		}()
		f.mux.ServeHTTP(rw, r)
	}()
	return contentTypeHTTPIntegrationResponse, nil
}

func newLambdaFunction(mux http.Handler) *lambdaFunction {
	return &lambdaFunction{
		mux: mux,
	}
}

// InvokeMode is the mode that determines which API operation Lambda uses.
type InvokeMode string

const (
	// InvokeModeBuffered indicates that your function is invoked using the Invoke API operation.
	// Invocation results are available when the payload is complete.
	InvokeModeBuffered InvokeMode = "BUFFERED"

	// InvokeModeResponseStream indicates that your function is invoked using
	// the InvokeWithResponseStream API operation.
	// It enables your function to stream payload results as they become available.
	InvokeModeResponseStream InvokeMode = "RESPONSE_STREAM"
)

// Start starts the AWS Lambda function.
// The handler is typically nil, in which case the DefaultServeMux is used.
func Start(mux http.Handler, mode InvokeMode) error {
	api := os.Getenv("AWS_LAMBDA_RUNTIME_API")
	if mux == nil {
		mux = http.DefaultServeMux
	}
	f := newLambdaFunction(mux)
	c := newRuntimeAPIClient(api)
	switch mode {
	case InvokeModeBuffered:
		if err := c.start(context.Background(), f.lambdaHandler); err != nil {
			log.Println(err)
			return err
		}
	case InvokeModeResponseStream:
		if err := c.startStreaming(context.Background(), f.lambdaHandlerStreaming); err != nil {
			log.Println(err)
			return err
		}
	default:
		return fmt.Errorf("ridgenative: invalid InvokeMode: %s", mode)
	}
	return nil
}

// ListenAndServe starts HTTP server.
//
// If AWS_LAMBDA_RUNTIME_API environment value is defined, it wait for new AWS Lambda events and handle it as HTTP requests.
// The format of the events is compatible with Amazon API Gateway Lambda proxy integration and Application Load Balancers.
// See AWS documents for details.
//
// https://docs.aws.amazon.com/apigateway/latest/developerguide/set-up-lambda-proxy-integrations.html
//
// https://docs.aws.amazon.com/elasticloadbalancing/latest/application/lambda-functions.html
//
// If AWS_EXECUTION_ENV environment value is AWS_Lambda_go1.x, it returns an error.
// If AWS_LAMBDA_RUNTIME_API environment value is NOT defined, it just calls http.ListenAndServe.
//
// The handler is typically nil, in which case the DefaultServeMux is used.
//
// If AWS_LAMBDA_RUNTIME_API environment value is defined, ListenAndServe uses it as the invoke mode.
// The default is InvokeModeBuffered.
func ListenAndServe(address string, mux http.Handler) error {
	if go1 := os.Getenv("AWS_EXECUTION_ENV"); go1 == "AWS_Lambda_go1.x" {
		// run on go1.x runtime
		return errors.New("ridgenative: go1.x runtime is not supported")
	}

	api := os.Getenv("AWS_LAMBDA_RUNTIME_API")
	if api == "" {
		// fall back to normal HTTP server.
		return http.ListenAndServe(address, mux)
	}

	// run on provided or provided.al2 runtime
	var mode InvokeMode
	switch os.Getenv("RIDGENATIVE_INVOKE_MODE") {
	case "BUFFERED", "":
		mode = InvokeModeBuffered
	case "RESPONSE_STREAM":
		mode = InvokeModeResponseStream
	default:
		return errors.New("ridgenative: invalid RIDGENATIVE_INVOKE_MODE")
	}
	return Start(mux, mode)
}
