package ridgenative

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
)

// invokeResponseError is the error response from the custom runtime.
type invokeResponseError struct {
	Message    string                           `json:"errorMessage"`
	Type       string                           `json:"errorType"`
	StackTrace []*invokeResponseErrorStackFrame `json:"stackTrace,omitempty"`
	ShouldExit bool                             `json:"-"`
}

func (e *invokeResponseError) Error() string {
	return fmt.Sprintf("%#v", e)
}

// invokeResponseErrorStackFrame is a single frame in the stack trace.
type invokeResponseErrorStackFrame struct {
	Path  string `json:"path"`
	Line  int    `json:"line"`
	Label string `json:"label"`
}

// panicInfo is the information about a panic.
type panicInfo struct {
	Message    string                           `json:"message"`    // Value passed to panic call, converted to string
	StackTrace []*invokeResponseErrorStackFrame `json:"stackTrace"` // Stack trace of the panic
}

// getPanicInfo returns the panic information.
func getPanicInfo(value any) panicInfo {
	message := fmt.Sprint(value)
	stack := getPanicStack()

	return panicInfo{Message: message, StackTrace: stack}
}

// defaultErrorFrameCount is the default number of frames to capture in the stack trace.
var defaultErrorFrameCount = 32

// getPanicStack returns the stack trace of the panic.
func getPanicStack() []*invokeResponseErrorStackFrame {
	s := make([]uintptr, defaultErrorFrameCount)
	const framesToHide = 3 // this (getPanicStack) -> getPanicInfo -> handler defer func
	n := runtime.Callers(framesToHide, s)
	if n == 0 {
		return []*invokeResponseErrorStackFrame{}
	}
	return convertStack(s[:n])
}

// convertStack converts a runtime stack trace into a slice of invokeResponseErrorStackFrame.
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

// formatFrame formats a runtime.Frame into an invokeResponseErrorStackFrame.
func formatFrame(inputFrame runtime.Frame) *invokeResponseErrorStackFrame {
	path := inputFrame.File
	line := inputFrame.Line
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

// getPanicInfo returns the type name of err.
func getErrorType(err any) string {
	errorType := reflect.TypeOf(err)
	if errorType.Kind() == reflect.Ptr {
		return errorType.Elem().Name()
	}
	return errorType.Name()
}

// lambdaPanicResponse returns the error response for a panic.
func lambdaPanicResponse(err any) *invokeResponseError {
	if ive, ok := err.(*invokeResponseError); ok {
		return ive
	}
	panicInfo := getPanicInfo(err)
	return &invokeResponseError{
		Message:    panicInfo.Message,
		Type:       getErrorType(err),
		StackTrace: panicInfo.StackTrace,
		ShouldExit: true,
	}
}

// lambdaErrorResponse returns the error response for a non-panic error.
func lambdaErrorResponse(invokeError error) *invokeResponseError {
	if ive, ok := invokeError.(*invokeResponseError); ok {
		return ive
	}
	return &invokeResponseError{
		Message: invokeError.Error(),
		Type:    getErrorType(invokeError),
	}
}
