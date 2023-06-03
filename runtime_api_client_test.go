package ridgenative

import (
	"context"
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
		w.Write([]byte(`{"key":"value"}`))
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
	t.Run("context deadline exceeded", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/2018-06-01/runtime/invocation/request-id/error" {
				t.Errorf("unexpected path: %s", r.URL.Path)
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
