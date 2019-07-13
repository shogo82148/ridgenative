package ridgenative

import (
	"context"
	"encoding/base64"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/textproto"
	"net/url"
	"path"
	"runtime"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type lambdaFunction struct {
	mux http.Handler
	buf [512]byte
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
}

func httpRequest(ctx context.Context, r request) (*http.Request, error) {
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
		contentLength = int64(len(r.Body))
		reader := io.Reader(strings.NewReader(r.Body))
		if r.IsBase64Encoded {
			// ignore padding
			for contentLength > 0 && r.Body[contentLength-1] == '=' {
				contentLength--
			}
			contentLength = contentLength * 3 / 4
			reader = base64.NewDecoder(base64.StdEncoding, reader)
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

type responseWriter struct {
	strings.Builder
	header      http.Header
	statusCode  int
	wroteHeader bool
	lambda      *lambdaFunction
}

type response struct {
	StatusCode        int                 `json:"statusCode"`
	StatusDescription string              `json:"statusDescription"`
	Headers           map[string]string   `json:"headers"`
	MultiValueHeaders map[string][]string `json:"multiValueHeaders"`
	Body              string              `json:"body"`
	IsBase64Encoded   bool                `json:"isBase64Encoded"`
}

func (f *lambdaFunction) newResponseWriter() *responseWriter {
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
}

func (rw *responseWriter) lambdaResponse() (response, error) {
	body := rw.Builder.String()
	contentType := rw.header.Get("Content-Type")
	if contentType == "" {
		copy(rw.lambda.buf[:], body)
		contentType = http.DetectContentType(rw.lambda.buf[:])
		rw.header.Set("Content-Type", contentType)
	}
	isBase64 := isBinary(contentType)
	if isBase64 {
		body = base64.StdEncoding.EncodeToString([]byte(body))
	}

	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
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
		IsBase64Encoded:   isBase64,
	}, nil
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
	r, err := httpRequest(ctx, req)
	if err != nil {
		return response{}, err
	}
	rw := f.newResponseWriter()
	f.mux.ServeHTTP(rw, r)
	return rw.lambdaResponse()
}

// Run runs http handler on Apex or net/http's server.
func Run(address string, mux http.Handler) {
	if mux == nil {
		mux = http.DefaultServeMux
	}
	f := &lambdaFunction{mux: mux}
	lambda.Start(f.lambdaHandler)
}
