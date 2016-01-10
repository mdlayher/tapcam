package main

import (
	"flag"
	"log"
	"net/http"
	"os"
)

var (
	host = flag.String("host", ":80", "host address for HTTP server")
	dir  = flag.String("d", "/tmp/tapcam", "directory for image storage")
)

func main() {
	flag.Parse()

	if err := os.MkdirAll(*dir, 0755); err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(*dir)))

	if err := http.ListenAndServe(*host, mux); err != nil {
		log.Fatal(err)
	}
}
