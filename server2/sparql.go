package server2

import (
	"github.com/mildred/SmartWeb/sparql"
	"net/http"
	"net/url"
	"io/ioutil"
	"strings"
	"bytes"
	"fmt"
	"io"
	"log"
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

func graphListCheck(allowedGraphs, graphList []string) ([]string, string) {
	if len(graphList) > 0 {
		for _, g := range graphList {
			found := false
			for _, ag := range allowedGraphs {
				if g == ag {
					found = true;
					break;
				}
			}
			if ! found {
				return []string{}, g
			}
		}
		return graphList, ""
	} else {
		return allowedGraphs, ""
	}
}

func listSubGraphs(dataSet *sparql.Client, u string) ([]string, error) {
	var allowedGraphs []string
	
	if strings.HasSuffix(u, "/") {
		result, err := dataSet.Select(sparql.MakeQuery(`
			SELECT DISTINCT ?g
			WHERE {
  				GRAPH ?g {?s ?p ?o}
				FILTER ( strstarts(str(?g), %1s) )
			}
		`, u));
		if err != nil {
			return []string{}, err
		}
		for _, row := range result.Results.Bindings {
			allowedGraphs = append(allowedGraphs, row["g"].Value)
		}
	} else {
		allowedGraphs = append(allowedGraphs, u)
	}
	
	return allowedGraphs, nil;
}

func (server SmartServer) handleSPARQLQuery(u *url.URL, res http.ResponseWriter, req *http.Request, vars url.Values) {
	// FIXME check that the sub graphs are allowed by ACL.
	
	allowedGraphs, err := listSubGraphs(server.dataSet, u.String());
	if err != nil {
		handleError(res, 500, err.Error())
		return
	}

	defaultGraphs, g1 := graphListCheck(allowedGraphs, vars["default-graph-uri"])	
	namedGraphs,   g2 := graphListCheck(allowedGraphs, vars["named-graph-uri"])

	if len(defaultGraphs) == 0 {
		handleError(res, 400, fmt.Sprintf("default-graph-uri not allowed for graph <%s>", g1))
		return
	}
	
	if len(namedGraphs) == 0 {
		handleError(res, 400, fmt.Sprintf("named-graph-uri not allowed for graph <%s>", g2))
		return
	}
	
	vars = url.Values{
		"query": []string{vars.Get("query")},
		"named-graph-uri": namedGraphs,
		"default-graph-uri": defaultGraphs,
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
	
	_, err = io.Copy(res, resp.Body);
	if err != nil {
		log.Println(err);
	}
}
