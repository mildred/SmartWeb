package server

import (
	"github.com/mildred/SmartWeb/sparql"
	"io"
	"strings"
	"log"
	"net/http"
	"net/url"
)

type DataSet interface {
	AddQuad(context, subject, predicate, object interface{}) error
	QueryGraph(query, content_type string) ([]byte, error)
}

var SmartWeb_hasReferer, _ = url.Parse("tag:mildred.fr,2015-05:SmartWeb#hasReferer")

type SmartServer struct {
	Root    Entry
	DataSet DataSet
	auth    Authenticator
}

func CreateFileServer(path string, dataSet DataSet) *SmartServer {
	return &SmartServer{
		Root:    CreateFSEntry(path),
		DataSet: dataSet,
		auth:    CreateAuthenticator(),
	}
}

func handleError(res http.ResponseWriter, status int, err string) {
	res.Header().Set("Content-Type", "text/plain, charset=utf-8")
	res.WriteHeader(http.StatusNotFound)
	res.Write([]byte(err))
}

func (server SmartServer) getEntry(Host string, Url *url.URL) Entry {
	if Host == "" {
		Host = "host"
	} else {
		Host = Host + ".host"
	}
	entry := server.Root.Child(Host).Child(Url.Path)
	meta_values := Url.Query()["meta"]
	for _, meta := range meta_values {
		entry = entry.Parameters().Child(meta)
	}
	return entry
}

func (server SmartServer) handleGET(entry Entry, res http.ResponseWriter, req *http.Request) {
	meta := entry.Parameters()

	f, err := entry.Open()
	if err != nil {
		handleError(res, http.StatusNotFound, err.Error())
		return
	}
	defer f.Close()
	
	var has_content_type bool = false

	if headers, err := meta.Child("headers").Children(); err == nil {
		for _, header := range headers {
			if data, err := header.Read(); err != nil {
				log.Println(err)
			} else {
				if string(header.Name()) == "Content-Type" {
					has_content_type = true
				}
				res.Header().Set(string(header.Name()), string(data))
			}
		}
	}
	
	if !has_content_type {

		var data [512]byte
		io.ReadFull(f, data[:]);
		f.Seek(0, 0)
		content_type := http.DetectContentType(data[:])
		
		if strings.HasPrefix(content_type, "text/plain") && strings.HasSuffix(req.URL.Path, ".css") {
			content_type = strings.Replace(content_type, "text/plain", "text/css", 1);
		}
		
		res.Header().Set("Content-Type", content_type)
	}

	res.WriteHeader(http.StatusOK)

	if req.Method != "HEAD" {
		io.Copy(res, f)
	}
}

func (server SmartServer) handlePUT(entry Entry, res http.ResponseWriter, req *http.Request) {
	meta := entry.Parameters()
	headers := meta.Child("headers/")
	exists := entry.Exists()

	err := headers.DeleteAll()
	if err != nil {
		handleError(res, http.StatusInternalServerError, err.Error())
		return
	}

	if contentType := req.Header.Get("Content-Type"); contentType != "" {
		err = headers.Child("Content-Type").Write([]byte(contentType))
		if err != nil {
			handleError(res, http.StatusInternalServerError, err.Error())
			return
		}
	}

	f, err := entry.Create()
	if err != nil {
		handleError(res, http.StatusInternalServerError, err.Error())
		return
	}
	defer f.Close()

	_, err = io.Copy(f, req.Body)
	if err != nil {
		handleError(res, http.StatusInternalServerError, err.Error())
		return
	}

	err = f.Commit()
	if err != nil {
		handleError(res, http.StatusInternalServerError, err.Error())
		return
	}

	if exists {
		res.WriteHeader(http.StatusNoContent)
	} else {
		res.WriteHeader(http.StatusCreated)
	}
}

func (server SmartServer) handleDELETE(entry Entry, res http.ResponseWriter, req *http.Request) {
	err := entry.DeleteAll()
	if err != nil {
		handleError(res, http.StatusInternalServerError, err.Error())
		return
	}

	res.WriteHeader(http.StatusNoContent)
}

func (server SmartServer) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	entry := server.getEntry(req.Host, req.URL)

	if !server.auth.Authenticate(entry, res, req) {
		res.WriteHeader(http.StatusUnauthorized)
		return
	}

	curUrl := (&url.URL{
		Scheme: "http", // Ignore https to avoid breaking links
		Host:   req.Host,
	}).ResolveReference(req.URL)

	_, has_rdf := req.URL.Query()["rdf"]
	if has_rdf {
		log.Println("RDF " + req.Method + " " + curUrl.String())
		if req.Method == "GET" {
			//server.DataSet
			var graphURL url.URL = *curUrl
			/*if graphURL.RawQuery == "rdf" {
				graphURL.RawQuery = ""
			} else {
				graphURL.RawQuery = strings.Replace("&"+graphURL.RawQuery+"&", "&rdf&", "&", -1)
				graphURL.RawQuery = graphURL.RawQuery[1 : len(graphURL.RawQuery)-1]
			}*/
			graph := sparql.IRILiteral(graphURL.String())
			q := `CONSTRUCT { ?s ?p ?o } WHERE { GRAPH ` + graph + ` { ?s ?p ?o } . }`
			log.Printf("Query %v\n", q)
			data, err := server.DataSet.QueryGraph(q, "text/turtle")
			res.Header().Set("Content-Type", "text/turtle")
			if err != nil {
				res.WriteHeader(http.StatusInternalServerError)
				res.Write([]byte(err.Error()))
			} else {
				res.WriteHeader(http.StatusOK)
				res.Write(data)
			}
		} else {
			res.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	log.Println(req.Method + " " + curUrl.String())

	referrer, err := url.Parse(req.Referer())
	if err == nil {
		referrer = curUrl.ResolveReference(referrer)
		var context url.URL = *curUrl
		if context.RawQuery == "" {
			context.RawQuery = "rdf"
		} else {
			context.RawQuery = context.RawQuery + "&rdf"
		}
		// Add the referrer in storage
		log.Printf("Add quad: %v %v %v %v\n", &context, curUrl, SmartWeb_hasReferer, referrer)
		err = server.DataSet.AddQuad(&context, curUrl, SmartWeb_hasReferer, referrer)
		if err != nil {
			log.Println(err)
		}
	}

	if req.Method == "GET" || req.Method == "HEAD" {
		server.handleGET(entry, res, req)
	} else if req.Method == "PUT" {
		server.handlePUT(entry, res, req)
	} else if req.Method == "DELETE" {
		server.handleDELETE(entry, res, req)
	} else {
		res.WriteHeader(http.StatusMethodNotAllowed)
	}
}
