package ridgenative

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"testing"
)

func loadRequest(path string) (request, error) {
	f, err := os.Open(path)
	if err != nil {
		return request{}, err
	}
	defer f.Close()

	var req request
	dec := json.NewDecoder(f)
	if err := dec.Decode(&req); err != nil {
		return request{}, err
	}
	return req, nil
}

func TestHTTPRequest(t *testing.T) {
	t.Run("alb get request", func(t *testing.T) {
		req, err := loadRequest("testdata/alb-get-request.json")
		if err != nil {
			t.Fatal(err)
		}
		httpReq, err := httpRequest(context.Background(), req)
		if err != nil {
			t.Fatal(err)
		}
		if httpReq.Method != http.MethodGet {
			t.Errorf("unexpected method: want %s, got %s", http.MethodGet, httpReq.Method)
		}
		if !reflect.DeepEqual(httpReq.Header["Header-Name"], []string{"Value1", "Value2"}) {
			t.Errorf("unexpected header: want %v, got %v", []string{"Value1", "Value2"}, httpReq.Header["Header-Name"])
		}
		if httpReq.RequestURI != "/foo/bar?query=hoge&query=fuga" {
			t.Errorf("unexpected RequestURI: want %q, got %q", "/foo/bar?query=hoge&query=fuga", httpReq.RequestURI)
		}
		if httpReq.ContentLength != 0 {
			t.Errorf("unexpected ContentLength: want %d, got %d", 0, httpReq.ContentLength)
		}
		body, err := ioutil.ReadAll(httpReq.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(body) != "" {
			t.Errorf("unexpected body: want %q, got %q", "", string(body))
		}
		if httpReq.Host != "lambda-test-1062019563.ap-northeast-1.elb.amazonaws.com" {
			t.Errorf("unexpected host: want %q, got %q", "lambda-test-1062019563.ap-northeast-1.elb.amazonaws.com", httpReq.Host)
		}
	})
}

func Benchmark(b *testing.B) {
	req, err := loadRequest("testdata/apigateway-base64-request.json")
	if err != nil {
		b.Fatal(err)
	}
	buf := make([]byte, 1024)
	for i := 0; i < b.N; i++ {
		r, _ := httpRequest(context.Background(), req)
		io.CopyBuffer(ioutil.Discard, r.Body, buf)
	}
}
