package main

import (
	"flag"
	"github.com/mildred/SmartWeb/server"
	"log"
	"net/http"
	"time"
)

func main() {
	var listen = flag.String("listen", ":8000", "Address to listen to")
	var path = flag.String("path", "./", "Path to serve")
	flag.Parse()

	srv := server.CreateFileServer(*path)

	s := &http.Server{
		Addr:           *listen,
		Handler:        srv,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	log.Fatal(s.ListenAndServe())
}
