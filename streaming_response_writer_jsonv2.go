//go:build go1.27

package ridgenative

import (
	"bufio"
	"encoding/json/v2"
	"fmt"
	"io"
	"log"
	"net/http"
	"path"
	"strings"
)

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
