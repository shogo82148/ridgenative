package ridgenative_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/shogo82148/ridgenative"
)

func ExampleListenAndServe() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintln(w, "Hello World")
	})
	go ridgenative.ListenAndServe(":8080", nil)
	time.Sleep(time.Second) // wait for starting the server.

	resp, err := http.Get("http://localhost:8080")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))

	// Output:
	// Hello World
}
