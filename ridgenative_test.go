package ridgenative

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"reflect"
	"testing"
)

func loadRequest(path string) (*request, error) {
	f, err := os.Open(path)
	if err != nil {
		return &request{}, err
	}
	defer f.Close()

	var req request
	dec := json.NewDecoder(f)
	if err := dec.Decode(&req); err != nil {
		return &request{}, err
	}
	return &req, nil
}

func TestHTTPRequest(t *testing.T) {
	l := newLambdaFunction(nil)
	t.Run("alb get request", func(t *testing.T) {
		req, err := loadRequest("testdata/alb-get-request.json")
		if err != nil {
			t.Fatal(err)
		}
		httpReq, err := l.httpRequestV1(context.Background(), req)
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
		body, err := io.ReadAll(httpReq.Body)
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
		httpReq, err := l.httpRequestV1(context.Background(), req)
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
		body, err := io.ReadAll(httpReq.Body)
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
		httpReq, err := l.httpRequestV1(context.Background(), req)
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
		body, err := io.ReadAll(httpReq.Body)
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
		httpReq, err := l.httpRequestV1(context.Background(), req)
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
		body, err := io.ReadAll(httpReq.Body)
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
		httpReq, err := l.httpRequestV1(context.Background(), req)
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
		body, err := io.ReadAll(httpReq.Body)
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
		httpReq, err := l.httpRequestV1(context.Background(), req)
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
		body, err := io.ReadAll(httpReq.Body)
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

	t.Run("api gateway v2 request", func(t *testing.T) {
		req, err := loadRequest("testdata/apigateway-v2-get-request.json")
		if err != nil {
			t.Fatal(err)
		}
		httpReq, err := l.httpRequestV2(context.Background(), req)
		if err != nil {
			t.Fatal(err)
		}
		if httpReq.Method != http.MethodGet {
			t.Errorf("unexpected method: want %s, got %s", http.MethodGet, httpReq.Method)
		}
		if !reflect.DeepEqual(httpReq.Header["Header1"], []string{"value1,value2"}) {
			t.Errorf("unexpected header: want %v, got %v", []string{"value1,value2"}, httpReq.Header["Header1"])
		}
		if httpReq.RequestURI != "/my/path?parameter1=value1&parameter1=value2&parameter2=value" {
			t.Errorf("unexpected RequestURI: want %q, got %q", "/my/path?parameter1=value1&parameter1=value2&parameter2=value", httpReq.RequestURI)
		}
		if httpReq.ContentLength != 0 {
			t.Errorf("unexpected ContentLength: want %d, got %d", 0, httpReq.ContentLength)
		}
		body, err := io.ReadAll(httpReq.Body)
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

	t.Run("api gateway v2 post request", func(t *testing.T) {
		req, err := loadRequest("testdata/apigateway-v2-post-request.json")
		if err != nil {
			t.Fatal(err)
		}
		httpReq, err := l.httpRequestV2(context.Background(), req)
		if err != nil {
			t.Fatal(err)
		}
		if httpReq.Method != http.MethodPost {
			t.Errorf("unexpected method: want %s, got %s", http.MethodPost, httpReq.Method)
		}
		if httpReq.RequestURI != "/my/path" {
			t.Errorf("unexpected RequestURI: want %q, got %q", "/my/path", httpReq.RequestURI)
		}
		if httpReq.ContentLength != int64(len("{\"hello\":\"world\"}")) {
			t.Errorf("unexpected ContentLength: want %d, got %d", int64(len("{\"hello\":\"world\"}")), httpReq.ContentLength)
		}
		body, err := io.ReadAll(httpReq.Body)
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

	t.Run("api gateway v2 base64 request", func(t *testing.T) {
		req, err := loadRequest("testdata/apigateway-v2-base64-request.json")
		if err != nil {
			t.Fatal(err)
		}
		httpReq, err := l.httpRequestV2(context.Background(), req)
		if err != nil {
			t.Fatal(err)
		}
		if httpReq.Method != http.MethodPost {
			t.Errorf("unexpected method: want %s, got %s", http.MethodPost, httpReq.Method)
		}
		if httpReq.RequestURI != "/my/path" {
			t.Errorf("unexpected RequestURI: want %q, got %q", "/my/path", httpReq.RequestURI)
		}
		if httpReq.ContentLength != int64(len("{\"hello\":\"world\"}")) {
			t.Errorf("unexpected ContentLength: want %d, got %d", int64(len("{\"hello\":\"world\"}")), httpReq.ContentLength)
		}
		body, err := io.ReadAll(httpReq.Body)
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

	t.Run("function urls get request", func(t *testing.T) {
		req, err := loadRequest("testdata/function-urls-get-request.json")
		if err != nil {
			t.Fatal(err)
		}
		httpReq, err := l.httpRequestV2(context.Background(), req)
		if err != nil {
			t.Fatal(err)
		}
		if httpReq.Method != http.MethodGet {
			t.Errorf("unexpected method: want %s, got %s", http.MethodGet, httpReq.Method)
		}
		if !reflect.DeepEqual(httpReq.Header["Header1"], []string{"value1,value2"}) {
			t.Errorf("unexpected header: want %v, got %v", []string{"value1,value2"}, httpReq.Header["Header1"])
		}
		if httpReq.RequestURI != "/foo /bar?parameter1=value1&parameter1=value2&parameter2=value" {
			t.Errorf("unexpected RequestURI: want %q, got %q", "/foo /bar?parameter1=value1&parameter1=value2&parameter2=value", httpReq.RequestURI)
		}
		if httpReq.ContentLength != 0 {
			t.Errorf("unexpected ContentLength: want %d, got %d", 0, httpReq.ContentLength)
		}
		body, err := io.ReadAll(httpReq.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(body) != "" {
			t.Errorf("unexpected body: want %q, got %q", "", string(body))
		}
		if httpReq.Host != "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx.lambda-url.ap-northeast-1.on.aws" {
			t.Errorf("unexpected host: want %q, got %q", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx.lambda-url.ap-northeast-1.on.aws", httpReq.Host)
		}
	})

	t.Run("function urls post request", func(t *testing.T) {
		req, err := loadRequest("testdata/function-urls-post-request.json")
		if err != nil {
			t.Fatal(err)
		}
		httpReq, err := l.httpRequestV2(context.Background(), req)
		if err != nil {
			t.Fatal(err)
		}
		if httpReq.Method != http.MethodPost {
			t.Errorf("unexpected method: want %s, got %s", http.MethodPost, httpReq.Method)
		}
		if httpReq.RequestURI != "/my/path" {
			t.Errorf("unexpected RequestURI: want %q, got %q", "/my/path", httpReq.RequestURI)
		}
		if httpReq.ContentLength != int64(len("{\"hello\":\"world\"}")) {
			t.Errorf("unexpected ContentLength: want %d, got %d", int64(len("{\"hello\":\"world\"}")), httpReq.ContentLength)
		}
		body, err := io.ReadAll(httpReq.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(body) != "{\"hello\":\"world\"}" {
			t.Errorf("unexpected body: want %q, got %q", "{\"hello\":\"world\"}", string(body))
		}
		if httpReq.Host != "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx.lambda-url.ap-northeast-1.on.aws" {
			t.Errorf("unexpected host: want %q, got %q", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx.lambda-url.ap-northeast-1.on.aws", httpReq.Host)
		}
	})

	t.Run("function urls base64 request", func(t *testing.T) {
		req, err := loadRequest("testdata/function-urls-post-base64-request.json")
		if err != nil {
			t.Fatal(err)
		}
		httpReq, err := l.httpRequestV2(context.Background(), req)
		if err != nil {
			t.Fatal(err)
		}
		if httpReq.Method != http.MethodPost {
			t.Errorf("unexpected method: want %s, got %s", http.MethodPost, httpReq.Method)
		}
		if httpReq.RequestURI != "/my/path" {
			t.Errorf("unexpected RequestURI: want %q, got %q", "/my/path", httpReq.RequestURI)
		}
		if httpReq.ContentLength != int64(len("{\"hello\":\"world\"}")) {
			t.Errorf("unexpected ContentLength: want %d, got %d", int64(len("{\"hello\":\"world\"}")), httpReq.ContentLength)
		}
		body, err := io.ReadAll(httpReq.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(body) != "{\"hello\":\"world\"}" {
			t.Errorf("unexpected body: want %q, got %q", "{\"hello\":\"world\"}", string(body))
		}
		if httpReq.Host != "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx.lambda-url.ap-northeast-1.on.aws" {
			t.Errorf("unexpected host: want %q, got %q", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx.lambda-url.ap-northeast-1.on.aws", httpReq.Host)
		}
	})
}

func TestResponseV1(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		rw := newResponseWriter()
		// normal header fields
		rw.Header().Add("foo", "foo")

		// multi line header fields
		rw.Header().Add("bar", "bar1")
		rw.Header().Add("bar", "bar2")

		// cookie
		rw.Header().Add("Set-Cookie", "foo1=bar1")
		rw.Header().Add("Set-Cookie", "foo2=bar2")

		if _, err := io.WriteString(rw, "<!DOCTYPE html>\n"); err != nil {
			t.Error(err)
		}
		if _, err := rw.Write([]byte("<html><body>Hello!</body></html>")); err != nil {
			t.Error(err)
		}

		resp, err := rw.lambdaResponseV1()
		if err != nil {
			t.Error(err)
		}
		if resp.Headers["Foo"] != "foo" {
			t.Errorf("unexpected header: want %q, got %q", "foo", resp.Headers["Foo"])
		}
		if resp.Headers["Bar"] != "bar1, bar2" {
			t.Errorf("unexpected header: want %q, got %q", "bar1, bar2", resp.Headers["Bar"])
		}
		if resp.Headers["Set-Cookie"] != "foo1=bar1" {
			t.Errorf("unexpected header: want %q, got %q", "bar1, bar2", resp.Headers["Bar"])
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
	t.Run("set content-type", func(t *testing.T) {
		rw := newResponseWriter()
		rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if _, err := io.WriteString(rw, "<!DOCTYPE html>\n"); err != nil {
			t.Error(err)
		}
		if _, err := rw.Write([]byte("<html><body>Hello!</body></html>")); err != nil {
			t.Error(err)
		}

		resp, err := rw.lambdaResponseV1()
		if err != nil {
			t.Error(err)
		}

		// Content-Type is auto detected.
		if resp.Headers["Content-Type"] != "text/plain; charset=utf-8" {
			t.Errorf("unexpected header: want %q, got %q", "text/plain; charset=utf-8", resp.Headers["Content-Type"])
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

		resp, err := rw.lambdaResponseV1()
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
	t.Run("base64", func(t *testing.T) {
		rw := newResponseWriter()
		// 1x1 PNG image
		if _, err := io.WriteString(rw, "\x89\x50\x4e\x47\x0d\x0a\x1a\x0a\x00\x00\x00\x0d\x49\x48\x44\x52"); err != nil {
			t.Error(err)
		}
		if _, err := io.WriteString(rw, "\x00\x00\x00\x01\x00\x00\x00\x01\x08\x04\x00\x00\x00\xb5\x1c\x0c"); err != nil {
			t.Error(err)
		}
		if _, err := io.WriteString(rw, "\x02\x00\x00\x00\x0b\x49\x44\x41\x54\x08\xd7\x63\x60\x60\x00\x00"); err != nil {
			t.Error(err)
		}
		if _, err := io.WriteString(rw, "\x00\x03\x00\x01\x20\xd5\x94\xc7\x00\x00\x00\x00\x49\x45\x4e\x44"); err != nil {
			t.Error(err)
		}
		if _, err := io.WriteString(rw, "\xae\x42\x60\x82"); err != nil {
			t.Error(err)
		}

		resp, err := rw.lambdaResponseV1()
		if err != nil {
			t.Error(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("unexpected status code: want %d, got %d", http.StatusOK, resp.StatusCode)
		}
		if resp.Body != "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVQI12NgYAAAAAMAASDVlMcAAAAASUVORK5CYII=" {
			t.Errorf("unexpected body: want %q, got %q", "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVQI12NgYAAAAAMAASDVlMcAAAAASUVORK5CYII=", resp.Body)
		}
		if !resp.IsBase64Encoded {
			t.Error("unexpected IsBase64Encoded: want true, got false")
		}
	})
}

func TestResponseV2(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		rw := newResponseWriter()

		// normal header fields
		rw.Header().Add("foo", "foo")

		// multi line header fields
		rw.Header().Add("bar", "bar1")
		rw.Header().Add("bar", "bar2")

		// cookie
		rw.Header().Add("Set-Cookie", "foo1=bar1")
		rw.Header().Add("Set-Cookie", "foo2=bar2")

		if _, err := io.WriteString(rw, "<!DOCTYPE html>\n"); err != nil {
			t.Error(err)
		}
		if _, err := rw.Write([]byte("<html><body>Hello!</body></html>")); err != nil {
			t.Error(err)
		}

		resp, err := rw.lambdaResponseV2()
		if err != nil {
			t.Error(err)
		}

		// test headers
		if resp.Headers["Foo"] != "foo" {
			t.Errorf("unexpected header: want %q, got %q", "foo", resp.Headers["Foo"])
		}
		if resp.Headers["Bar"] != "bar1, bar2" {
			t.Errorf("unexpected header: want %q, got %q", "bar1, bar2", resp.Headers["Bar"])
		}
		if v, ok := resp.Headers["Set-Cookie"]; ok {
			t.Errorf("unexpected header: want None, got %q", v)
		}
		if got, want := resp.Cookies, []string{"foo1=bar1", "foo2=bar2"}; !reflect.DeepEqual(got, want) {
			t.Errorf("unexpected cookie: want %#v, got %#v", want, got)
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
	t.Run("set content-type", func(t *testing.T) {
		rw := newResponseWriter()
		rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if _, err := io.WriteString(rw, "<!DOCTYPE html>\n"); err != nil {
			t.Error(err)
		}
		if _, err := rw.Write([]byte("<html><body>Hello!</body></html>")); err != nil {
			t.Error(err)
		}

		resp, err := rw.lambdaResponseV1()
		if err != nil {
			t.Error(err)
		}

		// Content-Type is auto detected.
		if resp.Headers["Content-Type"] != "text/plain; charset=utf-8" {
			t.Errorf("unexpected header: want %q, got %q", "text/plain; charset=utf-8", resp.Headers["Content-Type"])
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

		resp, err := rw.lambdaResponseV1()
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
	t.Run("base64", func(t *testing.T) {
		rw := newResponseWriter()
		// 1x1 PNG image
		if _, err := io.WriteString(rw, "\x89\x50\x4e\x47\x0d\x0a\x1a\x0a\x00\x00\x00\x0d\x49\x48\x44\x52"); err != nil {
			t.Error(err)
		}
		if _, err := io.WriteString(rw, "\x00\x00\x00\x01\x00\x00\x00\x01\x08\x04\x00\x00\x00\xb5\x1c\x0c"); err != nil {
			t.Error(err)
		}
		if _, err := io.WriteString(rw, "\x02\x00\x00\x00\x0b\x49\x44\x41\x54\x08\xd7\x63\x60\x60\x00\x00"); err != nil {
			t.Error(err)
		}
		if _, err := io.WriteString(rw, "\x00\x03\x00\x01\x20\xd5\x94\xc7\x00\x00\x00\x00\x49\x45\x4e\x44"); err != nil {
			t.Error(err)
		}
		if _, err := io.WriteString(rw, "\xae\x42\x60\x82"); err != nil {
			t.Error(err)
		}

		resp, err := rw.lambdaResponseV1()
		if err != nil {
			t.Error(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("unexpected status code: want %d, got %d", http.StatusOK, resp.StatusCode)
		}
		if resp.Body != "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVQI12NgYAAAAAMAASDVlMcAAAAASUVORK5CYII=" {
			t.Errorf("unexpected body: want %q, got %q", "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVQI12NgYAAAAAMAASDVlMcAAAAASUVORK5CYII=", resp.Body)
		}
		if !resp.IsBase64Encoded {
			t.Error("unexpected IsBase64Encoded: want true, got false")
		}
	})
}

func BenchmarkRequest_binary(b *testing.B) {
	l := newLambdaFunction(nil)
	req, err := loadRequest("testdata/apigateway-base64-request.json")
	if err != nil {
		b.Fatal(err)
	}
	req.Body = base64.StdEncoding.EncodeToString(make([]byte, 1<<20))
	req.IsBase64Encoded = true
	buf := make([]byte, 1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r, _ := l.httpRequestV1(context.Background(), req)
		io.CopyBuffer(io.Discard, r.Body, buf)
	}
}

func BenchmarkRequest_text(b *testing.B) {
	l := newLambdaFunction(nil)
	req, err := loadRequest("testdata/apigateway-base64-request.json")
	if err != nil {
		b.Fatal(err)
	}
	data := make([]byte, 1<<20)
	for i := 0; i < len(data); i++ {
		data[i] = 'a'
	}
	req.Body = string(data)
	req.IsBase64Encoded = false
	buf := make([]byte, 1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r, _ := l.httpRequestV1(context.Background(), req)
		io.CopyBuffer(io.Discard, r.Body, buf)
	}
}

func BenchmarkResponse_binary(b *testing.B) {
	data := make([]byte, 1<<20) // 1MB: the maximum size of the response JSON in ALB
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rw := newResponseWriter()
		rw.Write(data)
		rw.lambdaResponseV1()
	}
}

func BenchmarkResponse_text(b *testing.B) {
	data := make([]byte, 1<<20) // 1MB: the maximum size of the response JSON in ALB
	for i := 0; i < len(data); i++ {
		data[i] = 'a'
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rw := newResponseWriter()
		rw.Write(data)
		rw.lambdaResponseV1()
	}
}

func TestLambdaHandlerStreaming(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		l := newLambdaFunction(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if _, err := io.WriteString(w, `{"hello":"world"}`); err != nil {
				t.Error(err)
			}
		}))
		r, w := io.Pipe()
		contentType, err := l.lambdaHandlerStreaming(context.Background(), &request{
			RequestContext: requestContext{
				HTTP: &requestContextHTTP{
					Path: "/",
				},
			},
		}, w)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := contentType, "application/vnd.awslambda.http-integration-response"; got != want {
			t.Errorf("unexpected content type: want %q, got %q", want, got)
		}
		data, err := io.ReadAll(r)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := string(data), "{\"statusCode\":200,\"headers\":{\"Content-Type\":\"application/json\"}}\x00\x00\x00\x00\x00\x00\x00\x00{\"hello\":\"world\"}"; got != want {
			t.Errorf("unexpected body: want %q, got %q", want, got)
		}
	})

	t.Run("WriteHeader", func(t *testing.T) {
		l := newLambdaFunction(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			// Writes to ResponseWriter are buffered,
			// so multiple writes to ResponseWriter become a single write to the pipe
			if _, err := io.WriteString(w, `{"hello":`); err != nil {
				t.Error(err)
			}
			if _, err := io.WriteString(w, `"world"}`); err != nil {
				t.Error(err)
			}
		}))
		r, w := io.Pipe()
		contentType, err := l.lambdaHandlerStreaming(context.Background(), &request{
			RequestContext: requestContext{
				HTTP: &requestContextHTTP{
					Path: "/",
				},
			},
		}, w)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := contentType, "application/vnd.awslambda.http-integration-response"; got != want {
			t.Errorf("unexpected content type: want %q, got %q", want, got)
		}

		// Reads and Writes on the pipe are matched one to one,
		// so we get only the header on first read.
		buf := make([]byte, 1024)
		n, err := r.Read(buf)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := string(buf[:n]), "{\"statusCode\":200,\"headers\":{\"Content-Type\":\"application/json\"}}\x00\x00\x00\x00\x00\x00\x00\x00"; got != want {
			t.Errorf("unexpected body: want %q, got %q", want, got)
		}

		// The second read gets the body.
		n, err = r.Read(buf)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := string(buf[:n]), "{\"hello\":\"world\"}"; got != want {
			t.Errorf("unexpected body: want %q, got %q", want, got)
		}

		// The third read gets EOF.
		n, err = r.Read(buf)
		if err != io.EOF {
			t.Errorf("unexpected error: want %v, got %v", io.EOF, err)
		}
		if n != 0 {
			t.Errorf("unexpected read size: want %d, got %d", 0, n)
		}
	})

	t.Run("flush", func(t *testing.T) {
		l := newLambdaFunction(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			f, ok := w.(http.Flusher)
			if !ok {
				t.Error("http.ResponseWriter doesn't implement http.Flusher")
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			if _, err := io.WriteString(w, `{"hello":`); err != nil {
				t.Error(err)
			}
			f.Flush()
			if _, err := io.WriteString(w, `"world"}`); err != nil {
				t.Error(err)
			}
		}))
		r, w := io.Pipe()
		contentType, err := l.lambdaHandlerStreaming(context.Background(), &request{
			RequestContext: requestContext{
				HTTP: &requestContextHTTP{
					Path: "/",
				},
			},
		}, w)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := contentType, "application/vnd.awslambda.http-integration-response"; got != want {
			t.Errorf("unexpected content type: want %q, got %q", want, got)
		}

		// Reads and Writes on the pipe are matched one to one,
		// so we get only the header on first read.
		buf := make([]byte, 1024)
		n, err := r.Read(buf)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := string(buf[:n]), "{\"statusCode\":200,\"headers\":{\"Content-Type\":\"application/json\"}}\x00\x00\x00\x00\x00\x00\x00\x00"; got != want {
			t.Errorf("unexpected body: want %q, got %q", want, got)
		}

		// The second read gets the half of the body.
		n, err = r.Read(buf)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := string(buf[:n]), "{\"hello\":"; got != want {
			t.Errorf("unexpected body: want %q, got %q", want, got)
		}

		// The third read gets the rest of the body.
		n, err = r.Read(buf)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := string(buf[:n]), "\"world\"}"; got != want {
			t.Errorf("unexpected body: want %q, got %q", want, got)
		}

		// The forth read gets EOF.
		n, err = r.Read(buf)
		if err != io.EOF {
			t.Errorf("unexpected error: want %v, got %v", io.EOF, err)
		}
		if n != 0 {
			t.Errorf("unexpected read size: want %d, got %d", 0, n)
		}
	})

	t.Run("detect content-type", func(t *testing.T) {
		l := newLambdaFunction(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, err := io.WriteString(w, `<html></html>`); err != nil {
				t.Error(err)
			}
		}))
		r, w := io.Pipe()
		contentType, err := l.lambdaHandlerStreaming(context.Background(), &request{
			RequestContext: requestContext{
				HTTP: &requestContextHTTP{
					Path: "/",
				},
			},
		}, w)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := contentType, "application/vnd.awslambda.http-integration-response"; got != want {
			t.Errorf("unexpected content type: want %q, got %q", want, got)
		}

		data, err := io.ReadAll(r)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := string(data), "{\"statusCode\":200,\"headers\":{\"Content-Type\":\"text/html; charset=utf-8\"}}\x00\x00\x00\x00\x00\x00\x00\x00<html></html>"; got != want {
			t.Errorf("unexpected body: want %q, got %q", want, got)
		}
	})
}

func TestIsBinary(t *testing.T) {
	tests := []struct {
		header http.Header
		want   bool
	}{
		// text/html but encoded as gzip
		{
			header: http.Header{
				"Content-Type":     []string{"text/html; charset=utf-8"},
				"Content-Encoding": []string{"gzip"},
			},
			want: true,
		},

		// custom content encoding for Lambda
		{
			header: http.Header{
				"Content-Type":                   []string{"image/svg"},
				"X-Lambda-Http-Content-Encoding": []string{"text"},
			},
			want: false,
		},

		// text/*
		{
			header: http.Header{
				"Content-Type": []string{"text/html"},
			},
			want: false,
		},
		{
			header: http.Header{
				"Content-Type": []string{"text/plain"},
			},
			want: false,
		},
		{
			header: http.Header{
				"Content-Type": []string{"text/xml"},
			},
			want: false,
		},

		// data formats that are encoded as text
		{
			header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			want: false,
		},
		{
			header: http.Header{
				"Content-Type": []string{"application/javascript"},
			},
			want: false,
		},
		{
			header: http.Header{
				"Content-Type": []string{"application/yaml"},
			},
			want: false,
		},
		{
			header: http.Header{
				"Content-Type": []string{"application/xml"},
			},
			want: false,
		},

		// custom media types that are encoded as text
		{
			header: http.Header{
				"Content-Type": []string{"application/foo+json"},
			},
			want: false,
		},
		{
			header: http.Header{
				"Content-Type": []string{"application/foo+yaml"},
			},
			want: false,
		},
		{
			header: http.Header{
				"Content-Type": []string{"application/foo+xml"},
			},
			want: false,
		},
		{
			header: http.Header{
				"Content-Type": []string{"application/foo+xml ; charset=utf8"},
			},
			want: false,
		},

		// common binary formats
		{
			header: http.Header{
				"Content-Type": []string{"application/octet-stream"},
			},
			want: true,
		},
		{
			header: http.Header{
				"Content-Type": []string{"image/jpeg"},
			},
			want: true,
		},
		{
			header: http.Header{
				"Content-Type": []string{"audio/mpeg"},
			},
			want: true,
		},
		{
			header: http.Header{
				"Content-Type": []string{"unknown-content-type"},
			},
			want: true,
		},
		{
			header: http.Header{
				"Content-Type": []string{""},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		got := isBinary(tt.header)
		if got != tt.want {
			t.Errorf("isBinary(%v) = %v, want %v", tt.header, got, tt.want)
		}
	}
}
