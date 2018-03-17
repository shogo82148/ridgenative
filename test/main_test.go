package main

import (
	"bytes"
	"fmt"
	"io"
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
