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
	ridgenative.Run(":8080", "/api", nil)
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintln(w, "Hello World")
}
```

## RELATED WORKS

- [fujiwara/ridge](https://github.com/fujiwara/ridge)
- [apex/gateway](https://github.com/apex/gateway)
