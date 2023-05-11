package ridgenative

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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

	invoke, err := client.next()
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
