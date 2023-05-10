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
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type invoke struct {
	id      string
	payload []byte
	headers http.Header
	client  *runtimeAPIClient
}

type handlerFunc func(ctx context.Context, req *request) (*response, error)

// startRuntimeAPILoop will return an error if handling a particular invoke resulted in a non-recoverable error
func startRuntimeAPILoop(api string, h handlerFunc) error {
	client := newRuntimeAPIClient(api)
	for {
		invoke, err := client.next()
		if err != nil {
			return err
		}
		if err = handleInvoke(invoke, h); err != nil {
			return err
		}
	}
}

// handleInvoke returns an error if the function panics, or some other non-recoverable error occurred
func handleInvoke(invoke *invoke, h handlerFunc) error {
	// set the deadline
	deadline, err := parseDeadline(invoke)
	if err != nil {
		return reportFailure(invoke, lambdaErrorResponse(err))
	}
	ctx, cancel := context.WithDeadline(context.TODO(), deadline)
	defer cancel()

	// TODO: set the invoke metadata values

	// set the trace id
	traceID := invoke.headers.Get(headerTraceID)
	os.Setenv("_X_AMZN_TRACE_ID", traceID)
	// nolint:staticcheck
	ctx = context.WithValue(ctx, "x-amzn-trace-id", traceID)

	// TODO: call the handler, marshal any returned error
	response, invokeErr := callBytesHandlerFunc(ctx, invoke.payload, h)
	if invokeErr != nil {
		if err := reportFailure(invoke, invokeErr); err != nil {
			return err
		}
		if invokeErr.ShouldExit {
			return fmt.Errorf("calling the handler function resulted in a panic, the process should exit")
		}
		return nil
	}
	// if the response needs to be closed (ex: net.Conn, os.File), ensure it's closed before the next invoke to prevent a resource leak
	if response, ok := response.(io.Closer); ok {
		defer response.Close()
	}

	// if the response defines a content-type, plumb it through
	contentType := contentTypeBytes
	type ContentType interface{ ContentType() string }
	if response, ok := response.(ContentType); ok {
		contentType = response.ContentType()
	}

	if err := invoke.success(response, contentType); err != nil {
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

type invokeResponseError struct {
	Message    string                           `json:"errorMessage"`
	Type       string                           `json:"errorType"`
	StackTrace []*invokeResponseErrorStackFrame `json:"stackTrace,omitempty"`
	ShouldExit bool                             `json:"-"`
}

type invokeResponseErrorStackFrame struct {
	Path  string `json:"path"`
	Line  int32  `json:"line"`
	Label string `json:"label"`
}

func (e invokeResponseError) Error() string {
	return fmt.Sprintf("%#v", e)
}

func reportFailure(invoke *invoke, invokeErr *invokeResponseError) error {
	errorPayload := mustMarshal(invokeErr)
	log.Printf("%s", errorPayload)
	if err := invoke.failure(bytes.NewReader(errorPayload), contentTypeJSON); err != nil {
		return fmt.Errorf("unexpected error occurred when sending the function error to the API: %v", err)
	}
	return nil
}

func mustMarshal(v interface{}) []byte {
	payload, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return payload
}

func getErrorType(err interface{}) string {
	errorType := reflect.TypeOf(err)
	if errorType.Kind() == reflect.Ptr {
		return errorType.Elem().Name()
	}
	return errorType.Name()
}

func lambdaPanicResponse(err interface{}) *invokeResponseError {
	if ive, ok := err.(invokeResponseError); ok {
		return &ive
	}
	panicInfo := getPanicInfo(err)
	return &invokeResponseError{
		Message:    panicInfo.Message,
		Type:       getErrorType(err),
		StackTrace: panicInfo.StackTrace,
		ShouldExit: true,
	}
}

type panicInfo struct {
	Message    string                           `json:"message"`    // Value passed to panic call, converted to string
	StackTrace []*invokeResponseErrorStackFrame `json:"stackTrace"` // Stack trace of the panic
}

func getPanicInfo(value interface{}) panicInfo {
	message := getPanicMessage(value)
	stack := getPanicStack()

	return panicInfo{Message: message, StackTrace: stack}
}

func getPanicMessage(value interface{}) string {
	return fmt.Sprintf("%v", value)
}

var defaultErrorFrameCount = 32

func getPanicStack() []*invokeResponseErrorStackFrame {
	s := make([]uintptr, defaultErrorFrameCount)
	const framesToHide = 3 // this (getPanicStack) -> getPanicInfo -> handler defer func
	n := runtime.Callers(framesToHide, s)
	if n == 0 {
		return make([]*invokeResponseErrorStackFrame, 0)
	}

	s = s[:n]

	return convertStack(s)
}

func convertStack(s []uintptr) []*invokeResponseErrorStackFrame {
	var converted []*invokeResponseErrorStackFrame
	frames := runtime.CallersFrames(s)

	for {
		frame, more := frames.Next()

		formattedFrame := formatFrame(frame)
		converted = append(converted, formattedFrame)

		if !more {
			break
		}
	}
	return converted
}

func formatFrame(inputFrame runtime.Frame) *invokeResponseErrorStackFrame {
	path := inputFrame.File
	line := int32(inputFrame.Line)
	label := inputFrame.Function

	// Strip GOPATH from path by counting the number of separators in label & path
	//
	// For example given this:
	//     GOPATH = /home/user
	//     path   = /home/user/src/pkg/sub/file.go
	//     label  = pkg/sub.Type.Method
	//
	// We want to set:
	//     path  = pkg/sub/file.go
	//     label = Type.Method

	i := len(path)
	for n, g := 0, strings.Count(label, "/")+2; n < g; n++ {
		i = strings.LastIndex(path[:i], "/")
		if i == -1 {
			// Something went wrong and path has less separators than we expected
			// Abort and leave i as -1 to counteract the +1 below
			break
		}
	}

	path = path[i+1:] // Trim the initial /

	// Strip the path from the function name as it's already in the path
	label = label[strings.LastIndex(label, "/")+1:]
	// Likewise strip the package name
	label = label[strings.Index(label, ".")+1:]

	return &invokeResponseErrorStackFrame{
		Path:  path,
		Line:  line,
		Label: label,
	}
}

func lambdaErrorResponse(invokeError error) *invokeResponseError {
	if ive, ok := invokeError.(invokeResponseError); ok {
		return &ive
	}
	var errorName string
	if errorType := reflect.TypeOf(invokeError); errorType.Kind() == reflect.Ptr {
		errorName = errorType.Elem().Name()
	} else {
		errorName = errorType.Name()
	}
	return &invokeResponseError{
		Message: invokeError.Error(),
		Type:    errorName,
	}
}

func callBytesHandlerFunc(ctx context.Context, payload []byte, h handlerFunc) (response io.Reader, invokeErr *invokeResponseError) {
	defer func() {
		if err := recover(); err != nil {
			invokeErr = lambdaPanicResponse(err)
		}
	}()

	var req *request
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, lambdaErrorResponse(err)
	}
	resp, err := h(ctx, req)
	if err != nil {
		return nil, lambdaErrorResponse(err)
	}
	buf, err := json.Marshal(resp)
	if err != nil {
		return nil, lambdaErrorResponse(err)
	}
	response = bytes.NewReader(buf)
	return response, nil
}

// success sends the response payload for an in-progress invocation.
// Notes:
//   - An invoke is not complete until next() is called again!
func (i *invoke) success(body io.Reader, contentType string) error {
	url := i.client.baseURL + i.id + "/response"
	return i.client.post(url, body, contentType)
}

// failure sends the payload to the Runtime API. This marks the function's invoke as a failure.
// Notes:
//   - The execution of the function process continues, and is billed, until next() is called again!
//   - A Lambda Function continues to be re-used for future invokes even after a failure.
//     If the error is fatal (panic, unrecoverable state), exit the process immediately after calling failure()
func (i *invoke) failure(body io.Reader, contentType string) error {
	url := i.client.baseURL + i.id + "/error"
	return i.client.post(url, body, contentType)
}
