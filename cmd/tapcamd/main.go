package main

import (
	"bytes"
	"flag"
	"image"
	"image/jpeg"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/nfnt/resize"
)

var (
	host = flag.String("h", ":80", "host address for HTTP server")
	dir  = flag.String("d", "/tmp/tapcam", "directory for image storage")
)

func main() {
	flag.Parse()

	if err := os.MkdirAll(*dir, 0644); err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", &Handler{
		dir: *dir,
	})

	if err := http.ListenAndServe(*host, mux); err != nil {
		log.Fatal(err)
	}
}

type Handler struct {
	dir string
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	width := q.Get("width")
	if width == "" {
		http.FileServer(http.Dir(h.dir)).ServeHTTP(w, r)
		return
	}

	iwidth, err := strconv.Atoi(width)
	if err != nil {
		http.Error(w, "bad width value", http.StatusBadRequest)
		return
	}

	f, err := os.Open(filepath.Join(
		h.dir,
		filepath.Base(r.URL.Path),
	))
	if err != nil {
		log.Println(err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer func() { _ = f.Close() }()

	img, _, err := image.Decode(f)
	if err != nil {
		log.Println(err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	img = resize.Resize(uint(iwidth), 0, img, resize.NearestNeighbor)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		log.Println(err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/jpeg")

	_, _ = buf.WriteTo(w)
}
