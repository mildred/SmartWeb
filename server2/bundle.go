package server2

import (
	"github.com/mildred/SmartWeb/bundle"
	"github.com/mildred/SmartWeb/nquads"
	"github.com/mildred/SmartWeb/sparql"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"fmt"
	"strings"
)

func (server SmartServer) handlePOSTBundle(u *url.URL, res http.ResponseWriter, req *http.Request) {
	f, err := ioutil.TempFile(server.Root, "temp:")
	if err != nil {
		handleError(res, 500, err.Error())
		return
	}
	defer f.Close()
	
	_, err = io.Copy(f, req.Body)
	if err != nil {
		handleError(res, 500, err.Error())
		return
	}
	
	_, err = f.Seek(0, 0)
	if err != nil {
		handleError(res, 500, err.Error())
		return
	}
	
	if req.Header.Get("Content-Type") != bundle.MimeType {
		handleError(res, 400, fmt.Sprintf("Expected payload with type %s", bundle.MimeType))
		return
	}
	
	b, err := bundle.NewReader(f, req.ContentLength)
	if err != nil {
		handleError(res, 500, err.Error())
		return
	}
	
	statements, logs, err := makeStatements(u, b.GraphStatements(0))
	if err != nil {
		handleError(res, 400, err.Error())
		return
	}
	
	res.Write([]byte(statements))
	res.Write([]byte("\n"))
	res.Write([]byte(strings.Join(logs, "\n")))
}

var RdfNamespace   = "http://www.w3.org/1999/02/22-rdf-syntax-ns#"
var RdfType        = "http://www.w3.org/1999/02/22-rdf-syntax-ns#type"
var SwRelativePath = "tag:mildred.fr,2015-05:SmartWeb#relativePath"
var SwHash         = "tag:mildred.fr,2015-05:SmartWeb#hash"

func isSubUrl(base, u *url.URL) bool {
	if base.Scheme != u.Scheme ||
	   base.Opaque != u.Opaque ||
	   base.User   != u.User ||
	   base.Host   != u.Host {
		return false
	}
	if base.Path == u.Path || len(base.Path) == 0 {
		return true
	}
	basePath := base.Path
	if basePath[len(basePath)-1] != '/' {
		basePath += "/"
	}
	return strings.HasPrefix(u.Path, basePath)
}

func makeStatements(baseUri *url.URL, ch <-chan interface{}) (string, []string, error) {
	graphsRelUri := make(map[string]*url.URL)
	var statements string
	var logs []string
	for value := range ch {
		st, is_st := value.(*nquads.Statement)
		if !is_st {
			return "", logs, value.(error)
		}
		
		graph, has_graph := st.Graph()
		if ! has_graph && st.Predicate() == SwRelativePath {
			graph, _ := st.Subject()
			relUri, _, _, _ := st.ObjectLiteral()
			
			//graphUri := baseUri.ResolveReference(url.Parse(relUri))
			graphUri, err := baseUri.Parse(relUri)
			if err != nil {
				logs = append(logs, fmt.Sprintf(
					"Could not insert graph <%s>, its URI <%> is cannot be parsed: %s",
					graph, relUri, err.Error()))
				continue
			}
			if ! isSubUrl(baseUri, graphUri) {
				logs = append(logs, fmt.Sprintf(
					"Could not insert graph <%s>, its URI <%> is outside of out base <%s>",
					graph, relUri, baseUri.String()))
				continue
			}
			graphsRelUri[graph] = graphUri
			statements += sparql.MakeQuery(
				"DROP SILENT GRAPH %1u\n",
				graphUri)
		} else if graphUri, ok := graphsRelUri[graph]; has_graph && ok && graphUri != nil {
			var s, p, o string
			switch s_s, s_t := st.Subject(); s_t {
				default: continue
				case nquads.TypeIri:	s = sparql.IRILiteral(s_s); break
				case nquads.TypeBlank:	s = sparql.BlankLiteral(s_s); break
			}
			p = sparql.IRILiteral(st.Predicate())
			switch st.ObjectType() {
				default: continue
				case nquads.TypeIri:
					iri, _ := st.ObjectIri()
					o = sparql.IRILiteral(iri)
					break
				case nquads.TypeBlank:
					b, _ := st.ObjectBlank()
					o = sparql.BlankLiteral(b)
					break
				case nquads.TypeLiteral:
					val, typ, lang, _ := st.ObjectLiteral()
					if lang != "" {
						o = sparql.LocStringLiteral(val, lang)
					} else if typ != nquads.XsdString && typ != "" {
						o = sparql.TypedStringLiteral(val, typ)
					} else {
						o = sparql.StringLiteral(val)
					}
					break
			}
			statements += sparql.MakeQuery(
				"INSERT DATA { GRAPH %1u { %2q %3q %4q } }\n",
				graphUri, s, p, o)
		} else {
			if ! has_graph {
				logs = append(logs, fmt.Sprintf(
					"Could not insert %s\nPredicate %s != %s", st.String(), st.Predicate(), SwRelativePath))
			} else {
				logs = append(logs, fmt.Sprintf(
					"Could not insert %s", st.String()))
			}
			
		}
	}
	return statements, logs, nil
}
