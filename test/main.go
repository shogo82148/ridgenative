package main

import (
	"fmt"
	"image"
	_ "image/png"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/shogo82148/ridgenative"
)

func main() {
	http.HandleFunc("/get", handleGet)
	http.HandleFunc("/post/image", handlePostImage)
	ridgenative.Run(":8080", "/test", nil)
}

func handleGet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprint(w, "Hello ", r.FormValue("name"))
}

func handlePostImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Allowed POST method only", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	reader, err := r.MultipartReader()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for {
		p, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		width, height, err := parseImage(p)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "%d x %d\n", width, height)
	}
}

func parseImage(p *multipart.Part) (int, int, error) {
	img, _, err := image.Decode(p)
	if err != nil {
		return 0, 0, err
	}
	return img.Bounds().Dx(), img.Bounds().Dy(), nil
}
