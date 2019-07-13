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
	t.Run("alb post request", func(t *testing.T) {
		req, err := loadRequest("testdata/alb-post-request.json")
		if err != nil {
			t.Fatal(err)
		}
		httpReq, err := httpRequest(context.Background(), req)
		if err != nil {
			t.Fatal(err)
		}
		if httpReq.Method != http.MethodPost {
			t.Errorf("unexpected method: want %s, got %s", http.MethodPost, httpReq.Method)
		}
		if httpReq.RequestURI != "/" {
			t.Errorf("unexpected RequestURI: want %q, got %q", "/", httpReq.RequestURI)
		}
		if httpReq.ContentLength != int64(len("{\"hello\":\"world\"}")) {
			t.Errorf("unexpected ContentLength: want %d, got %d", int64(len("{\"hello\":\"world\"}")), httpReq.ContentLength)
		}
		body, err := ioutil.ReadAll(httpReq.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(body) != "{\"hello\":\"world\"}" {
			t.Errorf("unexpected body: want %q, got %q", "{\"hello\":\"world\"}", string(body))
		}
		if httpReq.Host != "lambda-test-1234567890.ap-northeast-1.elb.amazonaws.com" {
			t.Errorf("unexpected host: want %q, got %q", "lambda-test-1234567890.ap-northeast-1.elb.amazonaws.com", httpReq.Host)
		}
	})
	t.Run("alb bse64 request", func(t *testing.T) {
		req, err := loadRequest("testdata/alb-base64-request.json")
		if err != nil {
			t.Fatal(err)
		}
		httpReq, err := httpRequest(context.Background(), req)
		if err != nil {
			t.Fatal(err)
		}
		if httpReq.Method != http.MethodPost {
			t.Errorf("unexpected method: want %s, got %s", http.MethodPost, httpReq.Method)
		}
		if httpReq.RequestURI != "/foo/bar" {
			t.Errorf("unexpected RequestURI: want %q, got %q", "/foo/bar", httpReq.RequestURI)
		}
		if httpReq.ContentLength != int64(len("{\"hello\":\"world\"}")) {
			t.Errorf("unexpected ContentLength: want %d, got %d", int64(len("{\"hello\":\"world\"}")), httpReq.ContentLength)
		}
		body, err := ioutil.ReadAll(httpReq.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(body) != "{\"hello\":\"world\"}" {
			t.Errorf("unexpected body: want %q, got %q", "{\"hello\":\"world\"}", string(body))
		}
		if httpReq.Host != "lambda-test-1234567890.ap-northeast-1.elb.amazonaws.com" {
			t.Errorf("unexpected host: want %q, got %q", "lambda-test-1234567890.ap-northeast-1.elb.amazonaws.com", httpReq.Host)
		}
	})
	t.Run("api gateway get request", func(t *testing.T) {
		req, err := loadRequest("testdata/apigateway-get-request.json")
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
		if httpReq.RequestURI != "/foo%20/bar?query=hoge&query=fuga" {
			t.Errorf("unexpected RequestURI: want %q, got %q", "/foo%20/bar?query=hoge&query=fuga", httpReq.RequestURI)
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
		if httpReq.Host != "xxxxxxxxxx.execute-api.ap-northeast-1.amazonaws.com" {
			t.Errorf("unexpected host: want %q, got %q", "xxxxxxxxxx.execute-api.ap-northeast-1.amazonaws.com", httpReq.Host)
		}
	})
	t.Run("api gateway post request", func(t *testing.T) {
		req, err := loadRequest("testdata/apigateway-post-request.json")
		if err != nil {
			t.Fatal(err)
		}
		httpReq, err := httpRequest(context.Background(), req)
		if err != nil {
			t.Fatal(err)
		}
		if httpReq.Method != http.MethodPost {
			t.Errorf("unexpected method: want %s, got %s", http.MethodPost, httpReq.Method)
		}
		if httpReq.RequestURI != "/" {
			t.Errorf("unexpected RequestURI: want %q, got %q", "/", httpReq.RequestURI)
		}
		if httpReq.ContentLength != int64(len("{\"hello\":\"world\"}")) {
			t.Errorf("unexpected ContentLength: want %d, got %d", int64(len("{\"hello\":\"world\"}")), httpReq.ContentLength)
		}
		body, err := ioutil.ReadAll(httpReq.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(body) != "{\"hello\":\"world\"}" {
			t.Errorf("unexpected body: want %q, got %q", "{\"hello\":\"world\"}", string(body))
		}
		if httpReq.Host != "xxxxxxxxxx.execute-api.ap-northeast-1.amazonaws.com" {
			t.Errorf("unexpected host: want %q, got %q", "xxxxxxxxxx.execute-api.ap-northeast-1.amazonaws.com", httpReq.Host)
		}
	})
	t.Run("api gateway base64 request", func(t *testing.T) {
		req, err := loadRequest("testdata/apigateway-base64-request.json")
		if err != nil {
			t.Fatal(err)
		}
		httpReq, err := httpRequest(context.Background(), req)
		if err != nil {
			t.Fatal(err)
		}
		if httpReq.Method != http.MethodPost {
			t.Errorf("unexpected method: want %s, got %s", http.MethodPost, httpReq.Method)
		}
		if httpReq.RequestURI != "/" {
			t.Errorf("unexpected RequestURI: want %q, got %q", "/", httpReq.RequestURI)
		}
		if httpReq.ContentLength != int64(len("{\"hello\":\"world\"}")) {
			t.Errorf("unexpected ContentLength: want %d, got %d", int64(len("{\"hello\":\"world\"}")), httpReq.ContentLength)
		}
		body, err := ioutil.ReadAll(httpReq.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(body) != "{\"hello\":\"world\"}" {
			t.Errorf("unexpected body: want %q, got %q", "{\"hello\":\"world\"}", string(body))
		}
		if httpReq.Host != "xxxxxxxxxx.execute-api.ap-northeast-1.amazonaws.com" {
			t.Errorf("unexpected host: want %q, got %q", "xxxxxxxxxx.execute-api.ap-northeast-1.amazonaws.com", httpReq.Host)
		}
	})
}

func TestResponse(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		rw := newResponseWriter()
		rw.Header().Add("foo", "foo")
		rw.Header().Add("bar", "bar1")
		rw.Header().Add("bar", "bar2")

		if _, err := io.WriteString(rw, "<!DOCTYPE html>\n"); err != nil {
			t.Error(err)
		}
		if _, err := rw.Write([]byte("<html><body>Hello!</body></html>")); err != nil {
			t.Error(err)
		}

		resp, err := rw.lambdaResponse()
		if err != nil {
			t.Error(err)
		}
		if resp.Headers["Foo"] != "foo" {
			t.Errorf("unexpected header: want %q, got %q", "foo", resp.Headers["Foo"])
		}
		if resp.Headers["Bar"] != "bar1" {
			t.Errorf("unexpected header: want %q, got %q", "foo", resp.Headers["Foo"])
		}
		if !reflect.DeepEqual(resp.MultiValueHeaders["Foo"], []string{"foo"}) {
			t.Errorf("unexpected header: want %#v, got %#v", []string{"foo"}, resp.MultiValueHeaders["Foo"])
		}
		if !reflect.DeepEqual(resp.MultiValueHeaders["Bar"], []string{"bar1", "bar2"}) {
			t.Errorf("unexpected header: want %#v, got %#v", []string{"bar1", "bar2"}, resp.MultiValueHeaders["Bar"])
		}

		// Content-Type is auto detected.
		if resp.Headers["Content-Type"] != "text/html; charset=utf-8" {
			t.Errorf("unexpected header: want %q, got %q", "text/html; charset=utf-8", resp.Headers["Content-Type"])
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("unexpected status code: want %d, got %d", http.StatusOK, resp.StatusCode)
		}
		if resp.Body != "<!DOCTYPE html>\n<html><body>Hello!</body></html>" {
			t.Errorf("unexpected body: want %q, got %q", "<!DOCTYPE html>\n<html><body>Hello!</body></html>", resp.Body)
		}
		if resp.IsBase64Encoded {
			t.Error("unexpected IsBase64Encoded: want false, got true")
		}
	})
	t.Run("redirect to example.com", func(t *testing.T) {
		rw := newResponseWriter()
		rw.Header().Add("location", "http://example.com/")
		rw.WriteHeader(http.StatusFound)
		if _, err := io.WriteString(rw, "<!DOCTYPE html>\n"); err != nil {
			t.Error(err)
		}
		if _, err := rw.Write([]byte("<html><body>Redirect to <a href=http://example.com/>example.com</a></body></html>")); err != nil {
			t.Error(err)
		}

		resp, err := rw.lambdaResponse()
		if err != nil {
			t.Error(err)
		}
		if resp.Headers["Location"] != "http://example.com/" {
			t.Errorf("unexpected header: want %q, got %q", "http://example.com/", resp.Headers["Foo"])
		}
		if resp.StatusCode != http.StatusFound {
			t.Errorf("unexpected status code: want %d, got %d", http.StatusFound, resp.StatusCode)
		}
		if resp.Body != "<!DOCTYPE html>\n<html><body>Redirect to <a href=http://example.com/>example.com</a></body></html>" {
			t.Errorf("unexpected body: want %q, got %q", "<!DOCTYPE html>\n<html><body>Redirect to <a href=http://example.com/>example.com</a></body></html>", resp.Body)
		}
		if resp.IsBase64Encoded {
			t.Error("unexpected IsBase64Encoded: want false, got true")
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
