package main

import (
	"fmt"
	"net/http"

	"github.com/shogo82148/ridgenative"
)

func main() {
	http.HandleFunc("/get", handleGet)
	ridgenative.Run(":8080", "/test", nil)
}

func handleGet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprint(w, "Hello ", r.FormValue("name"))
}
