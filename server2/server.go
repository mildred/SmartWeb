package server2

import (
	"github.com/mildred/SmartWeb/sparql"
	"regexp"
	"log"
	"net/http"
	"net/url"
)

var SmartWeb_hasReferer, _ = url.Parse("tag:mildred.fr,2015-05:SmartWeb#hasReferer")

type SmartServer struct {
	Root     string
	dataSet *sparql.Client
}

func CreateFileServer(path string, query, update string) *SmartServer {
	return &SmartServer{
		Root:    path,
		dataSet: sparql.NewClient(query, update),
	}
}

func handleError(res http.ResponseWriter, status int, err string) {
	res.Header().Set("Content-Type", "text/plain, charset=utf-8")
	res.WriteHeader(status)
	res.Write([]byte(err))
}

func (server SmartServer) handleGET(rdfUrl, u *url.URL, res http.ResponseWriter, req *http.Request) {
	result, err := server.dataSet.Select(sparql.MakeQuery(`
		PREFIX sw: <tag:mildred.fr,2015-05:SmartWeb#>
		SELECT ?hash ?type
		WHERE {
			%1u
				sw:hash        ?hash ;
				sw:contentType ?type .
		}
		LIMIT 1
	`, u))
	
	if err != nil {
		handleError(res, 500, err.Error())
		return
	}
	
	if len(result.Results.Bindings) < 1 {
		handleError(res, 400, "")
		return
	}

	binding := result.Results.Bindings[0]
	hash := binding["hash"]
	content_type := binding["type"]

	res.Header().Set("Content-Type", content_type.Value)
	res.Header().Set("Etag", hash.Value)
	res.WriteHeader(http.StatusOK)
	res.Write([]byte{})
	
	/*
	if req.Method != "HEAD" {
		io.Copy(res, f)
	}
	*/
}

func (server SmartServer) handlePUT(rdfUrl, u *url.URL, res http.ResponseWriter, req *http.Request) {
	_, err := server.dataSet.Update(sparql.MakeQuery(`
		PREFIX sw: <tag:mildred.fr,2015-05:SmartWeb#>
		
		CLEAR SILENT GRAPH %1u;
		INSERT DATA {
			GRAPH %1u {
				%2u
					sw:hash        %3u ;
					sw:contentType %4s .
			}
		}
	`, rdfUrl, u, "hash:unknown", req.Header.Get("Content-Type")))
	
	if err != nil {
		handleError(res, 500, err.Error())
		return
	}

	res.WriteHeader(http.StatusCreated)
	/*
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
	}*/
}
/*
func (server SmartServer) handleDELETE(entry Entry, res http.ResponseWriter, req *http.Request) {
	err := entry.DeleteAll()
	if err != nil {
		handleError(res, http.StatusInternalServerError, err.Error())
		return
	}

	res.WriteHeader(http.StatusNoContent)
}*/

func (server SmartServer) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	curUrl := (&url.URL{
		Scheme: "http", // Ignore https to avoid breaking links
		Host:   req.Host,
	}).ResolveReference(req.URL)

	var context url.URL = *curUrl
	if context.RawQuery == "" {
		context.RawQuery = "rdf"
	} else {
		context.RawQuery = context.RawQuery + "&rdf"
	}

	log.Println(req.Method + " " + curUrl.String())

	referrer, err := url.Parse(req.Referer())
	if err == nil {
		referrer = curUrl.ResolveReference(referrer)
		// Add the referrer in storage
		err = server.dataSet.AddQuad(&context, curUrl, SmartWeb_hasReferer, referrer)
		if err != nil {
			log.Println(err)
		}
	}

	if req.Method == "GET" || req.Method == "HEAD" {
		server.handleGET(&context, curUrl, res, req)
	} else if req.Method == "PUT" {
		server.handlePUT(&context, curUrl, res, req)
	//} else if req.Method == "DELETE" {
	//	server.handleDELETE(entry, res, req)
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
