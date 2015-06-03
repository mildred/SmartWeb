package server2

import (
	"net/http"
	"net/url"
	"io/ioutil"
	"bytes"
	"fmt"
	"io"
)

func (server SmartServer) handleGETSPARQLQuery(u *url.URL, res http.ResponseWriter, req *http.Request) {
	graphUri := *u;
	graphUri.RawQuery = ""
	
	server.handleSPARQLQuery(&graphUri, res, req, u.Query())
}

func (server SmartServer) handlePOSTSPARQLQuery(u *url.URL, res http.ResponseWriter, req *http.Request) {
	vars := u.Query()

	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		handleError(res, 500, err.Error())
		return
	}
	vars.Add("query", string(data))

	graphUri := *u;
	graphUri.RawQuery = ""
	
	server.handleSPARQLQuery(&graphUri, res, req, vars)
}

func (server SmartServer) handleSPARQLQuery(u *url.URL, res http.ResponseWriter, req *http.Request, vars url.Values) {
	if vars.Get("default-graph-uri") != "" {
		handleError(res, 400, "default-graph-uri not allowed")
		return
	}
	
	if vars.Get("named-graph-uri") != "" {
		handleError(res, 400, "named-graph-uri not allowed")
		return
	}
	
	vars = url.Values{
		"query": []string{vars.Get("query")},
		"named-graph-uri": []string{u.String()},
		"default-graph-uri": []string{u.String()},
	}
	
	sparql, err := http.NewRequest("POST", server.dataSet.QueryUrl, bytes.NewReader([]byte(vars.Encode())))
	if err != nil {
		handleError(res, 500, err.Error())
		return
	}
	
	sparql.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	sparql.Header.Add("Accept", req.Header.Get("Accept"))
	
	resp, err := server.dataSet.Do(sparql)
	
	if err != nil {
		handleError(res, 500, err.Error())
		return
	}
	defer resp.Body.Close();
	
	if resp.StatusCode < 200 || (resp.StatusCode >= 300 && resp.StatusCode < 400) {
		handleError(res, 500, fmt.Sprintf("Received status %d", resp.StatusCode))
		return
	}
	
	res.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	res.WriteHeader(resp.StatusCode)
	io.Copy(res, resp.Body)
}
