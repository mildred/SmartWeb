package main

import (
	"flag"
	"github.com/mildred/SmartWeb/httpmux"
	"github.com/mildred/SmartWeb/server"
	"github.com/mildred/SmartWeb/rdf"
	"log"
	"net"
	"path/filepath"
	"crypto/tls"
	"os"
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

	dataSet, err := rdf.CreateRedlandDataSet(filepath.Join(*path, "rdf"))
	if err != nil {
		log.Fatal(err)
		return
	}

	defer dataSet.Close()
	srv := server.CreateFileServer(*path, dataSet)

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
	
	keypath := filepath.Join(*path, "key.pem");
	certpath := filepath.Join(*path, "cert.pem");
	cert, err := tls.LoadX509KeyPair(certpath, keypath)
	var config *tls.Config
	if err == nil {
		config = httpmux.NewTLSConfig([]tls.Certificate{cert})
	} else {
		keyfile, err1 := os.Create(keypath)
		certfile, err2 := os.Create(certpath)
		if keyfile != nil  { defer keyfile.Close(); }
		if certfile != nil { defer certfile.Close(); }

		log.Println(err);
		log.Println("Generating 2048 bits RSA self signed certificate...")
	
		var certBytes []byte
		var keyBytes  []byte
		config, certBytes, keyBytes, err = httpmux.NewSelfSignedRSAConfig(2048)
		if err != nil {
			log.Fatalf("Error generating certificate: %v\n", err)
			return
		}
		
		if err1 == nil && err2 == nil {
			_, err1 = keyfile.Write(keyBytes);
			_, err2 = certfile.Write(certBytes);
		}
		if err1 != nil {
			log.Fatalf("%s: %v\n", keypath, err1);
		}
		if err2 != nil {
			log.Fatalf("%s: %v\n", certpath, err2);
		}
	}


	listener := httpmux.NewListenerConfig(tcpKeepAliveListener{ln.(*net.TCPListener)}, config)

	log.Printf("Listening on %s\n", s.Addr)

	err = s.Serve(listener)
	if err != nil {
		log.Fatal(err)
		return
	}
}
