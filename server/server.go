package server

import (
	"github.com/mildred/SmartWeb/sparql"
	"io"
	"io/ioutil"
	"strings"
	"regexp"
	"log"
	"net/http"
	"net/url"
	"mime"
)

type DataSet interface {
	AddQuad(context, subject, predicate, object interface{}) error
	QueryGraph(query, baseUri string, accept_types []string) (data []byte, content_type string, err error, status int)
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

func (server SmartServer) serveRDFGraph(res http.ResponseWriter, req *http.Request, curUrl *url.URL) {
	log.Printf("RDF GET <%s>\n", curUrl.String())
	var graphURL url.URL = *curUrl
	graph := sparql.IRILiteral(graphURL.String())
	q := `CONSTRUCT { ?s ?p ?o } WHERE { GRAPH ` + graph + ` { ?s ?p ?o } . }`
	data, content_type, err, status := server.DataSet.QueryGraph(q, curUrl.String(), splitHeader(req.Header.Get("Accept")))
	if err != nil {
		res.Header().Set("Content-Type", "text/plain")
		res.WriteHeader(status)
		res.Write([]byte(err.Error()))
	} else {
		res.Header().Set("Content-Type", content_type)
		res.Header().Add("Vary", "Accept")
		res.WriteHeader(status)
		res.Write(data)
	}
}

func (server SmartServer) serveRDFQuery(res http.ResponseWriter, req *http.Request, curUrl *url.URL, query string) {
	log.Printf("RDF QUERY <%s>: %s\n", curUrl.String(), query)
	data, content_type, err, status := server.DataSet.QueryGraph(query, curUrl.String(), splitHeader(req.Header.Get("Accept")))
	if err != nil {
		res.Header().Set("Content-Type", "text/plain")
		res.WriteHeader(status)
		res.Write([]byte(err.Error()))
	} else {
		res.Header().Set("Content-Type", content_type)
		res.Header().Add("Vary", "Accept")
		res.WriteHeader(status)
		res.Write(data)
	}
}

var query_regexp1, _ = regexp.Compile("&query=[^&]*")
var query_regexp2, _ = regexp.Compile("^query=[^&]*&")

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
		if req.Method == "GET" {
			if query, has_query := req.URL.Query()["query"]; has_query && len(query) == 1 {
				curUrl.RawQuery = query_regexp1.ReplaceAllString(curUrl.RawQuery, "")
				curUrl.RawQuery = query_regexp2.ReplaceAllString(curUrl.RawQuery, "")
				server.serveRDFQuery(res, req, curUrl, query[0])
			} else {
				server.serveRDFGraph(res, req, curUrl)
			}
		} else if req.Method == "POST" {
			content_type, _, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
			if err != nil {
				res.Header().Set("Content-Type", "text/plain; charset=utf-8")
				res.WriteHeader(http.StatusBadRequest)
				res.Write([]byte(err.Error()))
			} else if content_type == "application/x-www-form-urlencoded" {
				err = req.ParseForm()
				if err != nil {
					res.WriteHeader(http.StatusBadRequest)
					res.Write([]byte(err.Error()))
				} else if query, has_query := req.Form["query"]; has_query && len(query) == 1 {
					server.serveRDFQuery(res, req, curUrl, query[0])
				}
			} else if content_type == "application/sparql-query" {
				query, err := ioutil.ReadAll(req.Body)
				if err != nil {
					res.WriteHeader(http.StatusInternalServerError)
					res.Write([]byte(err.Error()))
				} else {
					server.serveRDFQuery(res, req, curUrl, string(query))
				}
			} else {
				res.WriteHeader(http.StatusUnsupportedMediaType)
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

var header_value_regexp, _ = regexp.Compile(`([^,"]*|"([^"\\]*|\\.)*")*`)

func splitHeader(header string) []string {
	var result []string;
	var start = 0
	loc := header_value_regexp.FindStringIndex(header[start:])
	for n := 0; n < 100 && loc != nil; n++ {
		if start == 0 || header[start-1] == ',' {
			result = append(result, header[start+loc[0]:start+loc[1]])
		} else {
			result[len(result)-1] = result[len(result)-1] + header[start+loc[0]:start+loc[1]]
		}
		if start+loc[1] < len(header) && header[start+loc[1]] == ',' {
			start = start + loc[1] + 1
		} else {
			start = start + loc[1]
		}
		if start >= len(header) {
			loc = nil
		} else {
			loc = header_value_regexp.FindStringIndex(header[start:])
		}
	}
	return result
}
