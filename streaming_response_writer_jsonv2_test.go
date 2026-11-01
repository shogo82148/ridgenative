//go:build go1.27

package ridgenative

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestLambdaHandlerStreaming_MarshalError(t *testing.T) {
	// encoding/json/v2 rejects invalid UTF-8 by default,
	// so building the prelude fails.
	// The body must not be written in place of the prelude.
	body := "{\"statusCode\":302,\"headers\":{\"Location\":\"https://example.com\"}}\x00\x00\x00\x00\x00\x00\x00\x00"

	t.Run("with content-type", func(t *testing.T) {
		l := newLambdaFunction(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("X-Invalid-Utf8", "\xff")
			if _, err := io.WriteString(w, body); err == nil {
				t.Error("want error, got nil")
			}
		}))
		r, w := io.Pipe()
		_, err := l.lambdaHandlerStreaming(context.Background(), &request{
			Version: "2.0",
			RequestContext: requestContext{
				HTTP: &requestContextHTTP{
					Path: "/",
				},
			},
		}, w)
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(r)
		if err == nil {
			t.Error("want error, got nil")
		}
		if len(data) != 0 {
			t.Errorf("unexpected body: want empty, got %q", data)
		}
	})

	t.Run("WriteHeader", func(t *testing.T) {
		l := newLambdaFunction(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("X-Invalid-Utf8", "\xff")
			w.WriteHeader(http.StatusOK)
			if _, err := io.WriteString(w, body); err == nil {
				t.Error("want error, got nil")
			}
		}))
		r, w := io.Pipe()
		_, err := l.lambdaHandlerStreaming(context.Background(), &request{
			Version: "2.0",
			RequestContext: requestContext{
				HTTP: &requestContextHTTP{
					Path: "/",
				},
			},
		}, w)
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(r)
		if err == nil {
			t.Error("want error, got nil")
		}
		if len(data) != 0 {
			t.Errorf("unexpected body: want empty, got %q", data)
		}
	})

	t.Run("detect content-type", func(t *testing.T) {
		l := newLambdaFunction(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Invalid-Utf8", "\xff")
			// the first 512 bytes are used for detecting content-type,
			// and the rest must not be written.
			if _, err := io.WriteString(w, strings.Repeat(" ", 512)+body); err == nil {
				t.Error("want error, got nil")
			}
		}))
		r, w := io.Pipe()
		_, err := l.lambdaHandlerStreaming(context.Background(), &request{
			Version: "2.0",
			RequestContext: requestContext{
				HTTP: &requestContextHTTP{
					Path: "/",
				},
			},
		}, w)
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(r)
		if err == nil {
			t.Error("want error, got nil")
		}
		if len(data) != 0 {
			t.Errorf("unexpected body: want empty, got %q", data)
		}
	})
}
