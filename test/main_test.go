package main

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"testing"
	"time"
)

func runtest(t *testing.T, endpoint string) {
	t.Run("get", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s/test/get?name=World", endpoint))
		if err != nil {
			t.Fatal("fail to get", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("want 200, got %d", resp.StatusCode)
		}
		var b bytes.Buffer
		io.Copy(&b, resp.Body)
		if b.String() != "Hello World" {
			t.Errorf("want Hello World, got %s", b.String())
		}
		if resp.Header.Get("Content-Type") != "text/plain" {
			t.Errorf("want text/plain, got %s", resp.Header.Get("Content-Type"))
		}
	})

	t.Run("post image", func(t *testing.T) {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		part, err := writer.CreateFormFile("file", "test.png")
		if err != nil {
			t.Fatal(err)
		}
		img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
		if err := png.Encode(part, img); err != nil {
			t.Fatal(err)
		}
		writer.Close()

		req, err := http.NewRequest("POST", fmt.Sprintf("%s/test/post/image", endpoint), &body)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", writer.FormDataContentType())
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("want 200, got %d", resp.StatusCode)
		}
		var b bytes.Buffer
		io.Copy(&b, resp.Body)
		if b.String() != "16 x 16\n" {
			t.Errorf("want 16 x 16, got %s", b.String())
		}
		if resp.Header.Get("Content-Type") != "text/plain" {
			t.Errorf("want text/plain, got %s", resp.Header.Get("Content-Type"))
		}
	})
}

func TestOnLocal(t *testing.T) {
	go main()
	time.Sleep(time.Second)
	runtest(t, "http://localhost:8080")
}

func TestOnAWS(t *testing.T) {
	host := os.Getenv("RIDGE_HOST")
	if host != "" {
		runtest(t, host)
	} else {
		t.Skip("RIDGE_HOST is missing.")
	}
}
