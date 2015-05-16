package main

import (
	"flag"
	"github.com/mildred/SmartWeb/httpmux"
	"github.com/mildred/SmartWeb/server2"
	"log"
	"net"
	"path/filepath"
	"crypto/tls"
	"os"
	"fmt"
	"net/http"
	"time"
	"crypto/x509"
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

type SparqlEndpoint struct {
	query string
	update string
}

func main() {
	var listen = flag.String("listen", ":8000", "Address to listen to")
	var path   = flag.String("path", "./web", "Path to store raw files")
	var sparql_query_url  = flag.String("sparql-query-url", "", "URL to query the RDF DataStore")
	var sparql_update_url = flag.String("sparql-update-url", "", "URL to update the RDF DataStore")
	var rdf4store_port    = flag.Int("4s-port", 8080, "4store HTTP gateway port to autodetect SPARQL endpoints")
	flag.Parse()
	
	var sparql SparqlEndpoint

	if *sparql_query_url == "" {
		sparql.query = fmt.Sprintf("http://127.0.0.1:%d/sparql/", *rdf4store_port)
	} else {
		sparql.query = *sparql_query_url
	}
	
	if *sparql_update_url == "" {
		sparql.update = fmt.Sprintf("http://127.0.0.1:%d/update/", *rdf4store_port)
	} else {
		sparql.update = *sparql_update_url
	}
	
	log.Printf("SPARQL Query endpoint %s\n", sparql.query)
	log.Printf("SPARQL Update endpoint %s\n", sparql.update)
	
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
	
	x509Cert, err := x509.ParseCertificate(config.Certificates[0].Certificate[0])
	if err != nil {
		log.Fatal(err)
		return
	}

	srv := server2.CreateFileServer(*path, x509Cert, config.Certificates[0].PrivateKey, sparql.query, sparql.update)

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

	listener := httpmux.NewListenerConfig(tcpKeepAliveListener{ln.(*net.TCPListener)}, config)

	log.Printf("Listening on %s\n", s.Addr)

	err = s.Serve(listener)
	if err != nil {
		log.Fatal(err)
		return
	}
}
