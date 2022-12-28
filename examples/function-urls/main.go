package main

import (
	"fmt"
	"net/http"

	"github.com/shogo82148/ridgenative"
)

func main() {
	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/hello", handleHello)
	ridgenative.ListenAndServe(":8080", nil)
}

func handleHello(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "Hello %s\n", r.FormValue("name"))
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintln(w, "Hello World")
	fmt.Fprintln(w, r.URL)
}
