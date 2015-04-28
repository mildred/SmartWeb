package main

import (
	"flag"
	"github.com/mildred/SmartWeb/httpmux"
	"github.com/mildred/SmartWeb/server"
	"log"
	"net"
	"net/http"
	"time"
)

type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}

func main() {
	var listen = flag.String("listen", ":8000", "Address to listen to")
	var path = flag.String("path", "./web", "Path to serve")
	flag.Parse()

	srv := server.CreateFileServer(*path)

	s := &http.Server{
		Addr:           *listen,
		Handler:        srv,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		log.Fatal(err)
		return
	}

	log.Println("Generating self signed certificate...")

	config, err := httpmux.NewSelfSignedRSAConfig(4096)
	if err != nil {
		log.Fatalf("Error generating certificate: %v\n", err)
		return
	}

	listener := httpmux.NewListenerConfig(tcpKeepAliveListener{ln.(*net.TCPListener)}, config)

	log.Printf("Listening on %s\n", s.Addr)

	err = s.Serve(listener)
	if err != nil {
		log.Fatal(err)
		return
	}
}
