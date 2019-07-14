[![Build Status](https://travis-ci.com/shogo82148/ridgenative.svg?branch=master)](https://travis-ci.com/shogo82148/ridgenative)
[![GoDoc](https://godoc.org/github.com/shogo82148/ridgenative?status.svg)](https://godoc.org/github.com/shogo82148/ridgenative)

# ridgenative
AWS Lambda HTTP Proxy integration event bridge to Go net/http.
[fujiwara/ridge](https://github.com/fujiwara/ridge) is a prior work, but it depends on [Apex](http://apex.run/).
I want same one that only depends on [aws/aws-lambda-go](https://github.com/aws/aws-lambda-go).

## SYNOPSIS

```go
package main

import (
	"fmt"
	"net/http"

	"github.com/shogo82148/ridgenative"
)

func main() {
	http.HandleFunc("/", handleRoot)
	ridgenative.ListenAndServe(":8080", nil)
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintln(w, "Hello World")
}
```

## RELATED WORKS

- [fujiwara/ridge](https://github.com/fujiwara/ridge)
- [apex/gateway](https://github.com/apex/gateway)
