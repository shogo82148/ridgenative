package ridgenative

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestRuntimeAPIClient_next(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/2018-06-01/runtime/invocation/next" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set(headerAWSRequestID, "request-id")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"key":"value"}`)); err != nil {
			t.Error(err)
		}
	}))
	defer ts.Close()

	address := strings.TrimPrefix(ts.URL, "http://")
	client := newRuntimeAPIClient(address)

	invoke, err := client.next(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if invoke.id != "request-id" {
		t.Errorf("want id is %s, got %s", "request-id", invoke.id)
	}
	if string(invoke.payload) != `{"key":"value"}` {
		t.Errorf("want payload is %s, got %s", `{"key":"value"}`, string(invoke.payload))
	}
}

func TestRuntimeAPIClient_handleInvoke(t *testing.T) {
	t.Run("succeeds", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/2018-06-01/runtime/invocation/request-id/response" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			if r.Header.Get("Content-Type") != "application/json" {
				t.Errorf("unexpected content type: %s", r.Header.Get("Content-Type"))
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if string(body) != `{"statusCode":200,"body":"{\"key\":\"value\"}"}` {
				t.Errorf("unexpected body: %s", string(body))
			}
			w.WriteHeader(http.StatusAccepted)
		}))
		defer ts.Close()

		address := strings.TrimPrefix(ts.URL, "http://")
		client := newRuntimeAPIClient(address)

		invoke := &invoke{
			id: "request-id",
			headers: map[string][]string{
				"Lambda-Runtime-Deadline-Ms": {
					// the deadline is 100ms
					encodeDeadline(time.Now().Add(100 * time.Millisecond)),
				},
				"Lambda-Runtime-Trace-Id": {"trace-id"},
			},
			payload: []byte(`{"httpMethod":"GET","path":"/"}`),
		}
		err := client.handleInvoke(context.Background(), invoke, func(ctx context.Context, req *request) (*response, error) {
			// test trace id
			traceID := ctx.Value("x-amzn-trace-id").(string)
			if traceID != "trace-id" {
				t.Errorf("want trace id is %s, got %s", "trace-id", traceID)
			}
			if req.HTTPMethod != "GET" {
				t.Errorf("want method is %s, got %s", "GET", req.HTTPMethod)
			}
			if req.Path != "/" {
				t.Errorf("want path is %s, got %s", "/", req.Path)
			}

			return &response{
				StatusCode: 200,
				Body:       `{"key":"value"}`,
			}, nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/2018-06-01/runtime/invocation/request-id/error" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			if r.Header.Get("Content-Type") != "application/json" {
				t.Errorf("unexpected content type: %s", r.Header.Get("Content-Type"))
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if string(body) != `{"errorMessage":"some errors","errorType":"myError"}` {
				t.Errorf("unexpected body: %s", string(body))
			}
			w.WriteHeader(http.StatusAccepted)
		}))
		defer ts.Close()

		address := strings.TrimPrefix(ts.URL, "http://")
		client := newRuntimeAPIClient(address)

		invoke := &invoke{
			id: "request-id",
			headers: map[string][]string{
				"Lambda-Runtime-Deadline-Ms": {
					// the deadline is 100ms
					encodeDeadline(time.Now().Add(100 * time.Millisecond)),
				},
				"Lambda-Runtime-Trace-Id": {"trace-id"},
			},
			payload: []byte(`{"httpMethod":"GET","path":"/"}`),
		}
		err := client.handleInvoke(context.Background(), invoke, func(ctx context.Context, req *request) (*response, error) {
			return nil, &myError{"some errors"}
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("panic", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/2018-06-01/runtime/invocation/request-id/error" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			if r.Header.Get("Content-Type") != "application/json" {
				t.Errorf("unexpected content type: %s", r.Header.Get("Content-Type"))
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// ignore stack traces because it has line numbers and it is not stable.
			if !strings.HasPrefix(string(body), `{"errorMessage":"some errors","errorType":"string","stackTrace":`) {
				t.Errorf("unexpected body: %s", string(body))
			}
			w.WriteHeader(http.StatusAccepted)
		}))
		defer ts.Close()

		address := strings.TrimPrefix(ts.URL, "http://")
		client := newRuntimeAPIClient(address)

		invoke := &invoke{
			id: "request-id",
			headers: map[string][]string{
				"Lambda-Runtime-Deadline-Ms": {
					// the deadline is 100ms
					encodeDeadline(time.Now().Add(100 * time.Millisecond)),
				},
				"Lambda-Runtime-Trace-Id": {"trace-id"},
			},
			payload: []byte(`{"httpMethod":"GET","path":"/"}`),
		}
		err := client.handleInvoke(context.Background(), invoke, func(ctx context.Context, req *request) (*response, error) {
			panic("some errors")
		})
		if err == nil {
			t.Error("want error, but got nil")
		}
	})

	t.Run("context deadline exceeded", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/2018-06-01/runtime/invocation/request-id/error" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			if r.Header.Get("Content-Type") != "application/json" {
				t.Errorf("unexpected content type: %s", r.Header.Get("Content-Type"))
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if string(body) != `{"errorMessage":"context deadline exceeded","errorType":"myError"}` {
				t.Errorf("unexpected body: %s", string(body))
			}
			w.WriteHeader(http.StatusAccepted)
		}))
		defer ts.Close()

		address := strings.TrimPrefix(ts.URL, "http://")
		client := newRuntimeAPIClient(address)

		invoke := &invoke{
			id: "request-id",
			headers: map[string][]string{
				"Lambda-Runtime-Deadline-Ms": {
					// the deadline is 100ms
					encodeDeadline(time.Now().Add(100 * time.Millisecond)),
				},
			},
			payload: []byte(`{}`),
		}
		err := client.handleInvoke(context.Background(), invoke, func(ctx context.Context, req *request) (*response, error) {
			select {
			// the handle takes a long time, so the deadline is exceeded.
			case <-time.After(time.Second):
				t.Error("deadline is too long")
				return nil, errors.New("timeout")

			case <-ctx.Done():
				return nil, &myError{"context deadline exceeded"}
			}
		})
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestRuntimeAPIClient_handleInvokeStreaming(t *testing.T) {
	t.Run("succeeds", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/2018-06-01/runtime/invocation/request-id/response" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			if r.Header.Get("Content-Type") != "application/vnd.awslambda.http-integration-response" {
				t.Errorf("unexpected content type: %s", r.Header.Get("Content-Type"))
			}
			if r.Header.Get("Lambda-Runtime-Function-Response-Mode") != "streaming" {
				t.Errorf("unexpected response mode: %s", r.Header.Get("Lambda-Runtime-Function-Response-Mode"))
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if string(body) != `{"statusCode":200,"body":"{\"key\":\"value\"}"}` {
				t.Errorf("unexpected body: %s", string(body))
			}
			if len(r.Trailer.Values("Lambda-Runtime-Function-Error-Type")) != 0 {
				t.Errorf("unexpected error type: %s", r.Trailer.Values("Lambda-Runtime-Function-Error-Type"))
			}
			if len(r.Trailer.Values("Lambda-Runtime-Function-Error-Body")) != 0 {
				t.Errorf("unexpected error body: %s", r.Trailer.Values("Lambda-Runtime-Function-Error-Body"))
			}
			w.WriteHeader(http.StatusAccepted)
		}))
		defer ts.Close()

		address := strings.TrimPrefix(ts.URL, "http://")
		client := newRuntimeAPIClient(address)

		invoke := &invoke{
			id: "request-id",
			headers: map[string][]string{
				"Lambda-Runtime-Deadline-Ms": {
					// the deadline is 100ms
					encodeDeadline(time.Now().Add(100 * time.Millisecond)),
				},
				"Lambda-Runtime-Trace-Id": {"trace-id"},
			},
			payload: []byte(`{"httpMethod":"GET","path":"/"}`),
		}
		err := client.handleInvokeStreaming(context.Background(), invoke, func(ctx context.Context, req *request, w *io.PipeWriter) (string, error) {
			traceID := ctx.Value("x-amzn-trace-id").(string)
			if traceID != "trace-id" {
				t.Errorf("want trace id is %s, got %s", "trace-id", traceID)
			}
			if req.HTTPMethod != "GET" {
				t.Errorf("want method is %s, got %s", "GET", req.HTTPMethod)
			}
			if req.Path != "/" {
				t.Errorf("want path is %s, got %s", "/", req.Path)
			}

			go func() {
				if _, err := io.WriteString(w, `{"statusCode":200,"body":"{\"key\":\"value\"}"}`); err != nil {
					t.Error(err)
				}
				if err := w.Close(); err != nil {
					t.Error(err)
				}
			}()

			return "application/vnd.awslambda.http-integration-response", nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("error before start streaming", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/2018-06-01/runtime/invocation/request-id/error" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			if r.Header.Get("Content-Type") != "application/json" {
				t.Errorf("unexpected content type: %s", r.Header.Get("Content-Type"))
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if string(body) != `{"errorMessage":"some errors","errorType":"myError"}` {
				t.Errorf("unexpected body: %s", string(body))
			}
			w.WriteHeader(http.StatusAccepted)
		}))
		defer ts.Close()

		address := strings.TrimPrefix(ts.URL, "http://")
		client := newRuntimeAPIClient(address)

		invoke := &invoke{
			id: "request-id",
			headers: map[string][]string{
				"Lambda-Runtime-Deadline-Ms": {
					// the deadline is 100ms
					encodeDeadline(time.Now().Add(100 * time.Millisecond)),
				},
				"Lambda-Runtime-Trace-Id": {"trace-id"},
			},
			payload: []byte(`{"httpMethod":"GET","path":"/"}`),
		}
		err := client.handleInvokeStreaming(context.Background(), invoke, func(ctx context.Context, req *request, w *io.PipeWriter) (string, error) {
			return "", &myError{"some errors"}
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("error during streaming", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/2018-06-01/runtime/invocation/request-id/response" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			if r.Header.Get("Content-Type") != "application/vnd.awslambda.http-integration-response" {
				t.Errorf("unexpected content type: %s", r.Header.Get("Content-Type"))
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if string(body) != "" {
				t.Errorf("unexpected body: %s", string(body))
			}

			if r.Trailer.Get("Lambda-Runtime-Function-Error-Type") != "myError" {
				t.Errorf("unexpected error type: %s", r.Trailer.Get("Lambda-Runtime-Function-Error-Type"))
			}
			wantErr := base64.StdEncoding.EncodeToString([]byte(`{"errorMessage":"some errors","errorType":"myError"}`))
			if r.Trailer.Get("Lambda-Runtime-Function-Error-Body") != wantErr {
				t.Errorf("unexpected error: %s", r.Trailer.Get("Lambda-Runtime-Function-Error-Body"))
			}

			w.WriteHeader(http.StatusAccepted)
		}))
		defer ts.Close()

		address := strings.TrimPrefix(ts.URL, "http://")
		client := newRuntimeAPIClient(address)

		invoke := &invoke{
			id: "request-id",
			headers: map[string][]string{
				"Lambda-Runtime-Deadline-Ms": {
					// the deadline is 100ms
					encodeDeadline(time.Now().Add(100 * time.Millisecond)),
				},
				"Lambda-Runtime-Trace-Id": {"trace-id"},
			},
			payload: []byte(`{"httpMethod":"GET","path":"/"}`),
		}
		err := client.handleInvokeStreaming(context.Background(), invoke, func(ctx context.Context, req *request, w *io.PipeWriter) (string, error) {
			go func() {
				if err := w.CloseWithError(&myError{"some errors"}); err != nil {
					t.Error(err)
				}
			}()

			return "application/vnd.awslambda.http-integration-response", nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("panic", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/2018-06-01/runtime/invocation/request-id/error" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			if r.Header.Get("Content-Type") != "application/json" {
				t.Errorf("unexpected content type: %s", r.Header.Get("Content-Type"))
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// ignore stack traces because it has line numbers and it is not stable.
			if !strings.HasPrefix(string(body), `{"errorMessage":"some errors","errorType":"string","stackTrace":`) {
				t.Errorf("unexpected body: %s", string(body))
			}
			w.WriteHeader(http.StatusAccepted)
		}))
		defer ts.Close()

		address := strings.TrimPrefix(ts.URL, "http://")
		client := newRuntimeAPIClient(address)

		invoke := &invoke{
			id: "request-id",
			headers: map[string][]string{
				"Lambda-Runtime-Deadline-Ms": {
					// the deadline is 100ms
					encodeDeadline(time.Now().Add(100 * time.Millisecond)),
				},
				"Lambda-Runtime-Trace-Id": {"trace-id"},
			},
			payload: []byte(`{"httpMethod":"GET","path":"/"}`),
		}
		err := client.handleInvokeStreaming(context.Background(), invoke, func(ctx context.Context, req *request, w *io.PipeWriter) (string, error) {
			panic("some errors")
		})
		if err == nil {
			t.Error("want error, but got nil")
		}
	})

	t.Run("context deadline exceeded", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/2018-06-01/runtime/invocation/request-id/error" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			if r.Header.Get("Content-Type") != "application/json" {
				t.Errorf("unexpected content type: %s", r.Header.Get("Content-Type"))
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Error(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if string(body) != `{"errorMessage":"context deadline exceeded","errorType":"myError"}` {
				t.Errorf("unexpected body: %s", string(body))
			}
			w.WriteHeader(http.StatusAccepted)
		}))
		defer ts.Close()

		address := strings.TrimPrefix(ts.URL, "http://")
		client := newRuntimeAPIClient(address)

		invoke := &invoke{
			id: "request-id",
			headers: map[string][]string{
				"Lambda-Runtime-Deadline-Ms": {
					// the deadline is 100ms
					encodeDeadline(time.Now().Add(100 * time.Millisecond)),
				},
				"Lambda-Runtime-Trace-Id": {"trace-id"},
			},
			payload: []byte(`{"httpMethod":"GET","path":"/"}`),
		}
		err := client.handleInvokeStreaming(context.Background(), invoke, func(ctx context.Context, req *request, w *io.PipeWriter) (string, error) {
			select {
			// the handle takes a long time, so the deadline is exceeded.
			case <-time.After(time.Second):
				t.Error("deadline is too long")
				return "", errors.New("timeout")

			case <-ctx.Done():
				return "", &myError{"context deadline exceeded"}
			}
		})
		if err != nil {
			t.Fatal(err)
		}
	})

}

type myError struct {
	msg string
}

func (e *myError) Error() string {
	return e.msg
}

func encodeDeadline(t time.Time) string {
	ms := t.UnixMilli()
	return strconv.FormatInt(ms, 10)
}
