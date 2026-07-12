//go:build !go1.27

package ridgenative

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// errorCapturingReader is a reader that captures the first error returned by the underlying reader.
type errorCapturingReader struct {
	reader  io.ReadCloser
	err     error
	trailer http.Header
}

func newErrorCapturingReader(r io.ReadCloser) *errorCapturingReader {
	return &errorCapturingReader{
		reader:  r,
		trailer: http.Header{},
	}
}

func (r *errorCapturingReader) Read(p []byte) (int, error) {
	if r.reader == nil {
		return 0, io.EOF
	}
	if r.err != nil {
		return 0, r.err
	}

	n, err := r.reader.Read(p)
	if err != nil && !errors.Is(err, io.EOF) {
		// capture the error
		lambdaErr := lambdaErrorResponse(err)
		body, err := json.Marshal(lambdaErr)
		if err != nil {
			// marshaling lambdaErr always succeeds
			// because lambdaErr doesn't have any functions and channels.
			panic(err)
		}
		r.trailer.Set(trailerLambdaErrorType, lambdaErr.Type)
		r.trailer.Set(trailerLambdaErrorBody, base64.StdEncoding.EncodeToString(body))
		r.err = io.EOF
		return n, io.EOF
	}
	return n, err
}

func (r *errorCapturingReader) Close() error {
	if r.reader == nil {
		return nil
	}
	return r.reader.Close()
}
