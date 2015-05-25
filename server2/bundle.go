package server2

import (
	"archive/zip"
	"github.com/mildred/SmartWeb/bundle"
	"github.com/mildred/SmartWeb/nquads"
	"github.com/mildred/SmartWeb/sparql"
	"os"
	"path"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"fmt"
	"strings"
	"crypto/sha1"
	"encoding/hex"
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
	
	statements, wantedHashes, logs, err := makeStatements(u, b.GraphStatements(0))
	if err != nil {
		handleError(res, 400, err.Error())
		return
	}
	
	for _, zipfile := range b.Reader.File {
		if strings.HasPrefix(zipfile.Name, "sha1:") {
			hash, err := getBundleSHA1(zipfile)
			if err != nil {
				handleError(res, 500, err.Error())
				return
			}
			if ! wantedHashes[hash] {
				continue
			}
			
			zf, err := zipfile.Open()
			if err != nil {
				handleError(res, 500, err.Error())
				return
			}
			defer zf.Close()
			
			f, err := ioutil.TempFile(server.Root, "temp:")
			if err != nil {
				handleError(res, 500, err.Error())
				return
			}
			defer f.Close()
			
			_, err = io.Copy(f, zf)
			if err != nil {
				go os.Remove(f.Name())
				handleError(res, 500, err.Error())
				return
			}
			
			err = os.Rename(f.Name(), path.Join(server.Root, hash))
			if err != nil {
				go os.Remove(f.Name())
				handleError(res, 500, err.Error())
				return
			}
		}
	}
	
	res.Write([]byte(statements))
	res.Write([]byte("\n"))
	res.Write([]byte(strings.Join(logs, "\n")))
}

func getBundleSHA1(zipfile *zip.File) (string, error) {
	h := sha1.New()
	f, err := zipfile.Open()
	if err != nil {
		return "", err
	}
	defer f.Close()
	_, err = io.Copy(h, f)
	if err != nil {
		return "", err
	}
	return "sha1:" + strings.ToLower(hex.EncodeToString(h.Sum([]byte{}))), nil
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

func makeStatements(baseUri *url.URL, ch <-chan interface{}) (string, map[string]bool, []string, error) {
	graphsRelUri := make(map[string]*url.URL)
	var ins inserter
	var logs []string
	var wantedHashes map[string]bool = make(map[string]bool)
	for value := range ch {
		st, is_st := value.(*nquads.Statement)
		if !is_st {
			return "", wantedHashes, logs, value.(error)
		}
		
		graph, has_graph := st.Graph()
		if ! has_graph && st.Predicate() == SwRelativePath {
			graph, _ := st.Subject()
			relUri, _, _, _ := st.ObjectLiteral()
			
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
			ins.deleteGraph(graphUri)
		} else if graphUri, ok := graphsRelUri[graph]; has_graph && ok && graphUri != nil {
			if st.Predicate() == SwHash {
				hash, is_hash := st.ObjectIri()
				if is_hash {
					wantedHashes[hash] = true
				}
			}
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
			ins.insertData(s, p, o, sparql.IRILiteral(graphUri.String()))
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
	statements := ins.terminate()
	return statements, wantedHashes, logs, nil
}


type inserter struct {
	statements string
	inDataStatement bool
	currentGraph string
	currentSubject string
	currentPredicate string
	inObject bool
}

func (i *inserter) closeObject(suffix string) {
	if i.inObject {
		i.statements += "," + suffix
		i.inObject = false
	}
}

func (i *inserter) closePredicate(suffix string) {
	if i.currentPredicate != "" {
		i.statements += " ;" + suffix
		i.currentPredicate = ""
		i.inObject = false
	}
}

func (i *inserter) closeSubject(suffix string) {
	if i.currentSubject != "" {
		i.statements += " ." + suffix
		i.currentSubject = ""
		i.currentPredicate = ""
		i.inObject = false
	}
}

func (i *inserter) closeGraph(suffix string) {
	if i.currentGraph != "" {
		i.closeSubject("");
		i.statements += " }" + suffix
		i.currentGraph = ""
	}
}

func (i *inserter) closeData(suffix string) {
	if i.inDataStatement {
		i.closeGraph("");
		i.statements += " }" + suffix
		i.inDataStatement = false
	}
}

func (i *inserter) deleteGraph(graphUri *url.URL) {
	i.closeData("\n")
	i.statements += sparql.MakeQuery(
		"DROP SILENT GRAPH %1u\n",
		graphUri)
}

func (i *inserter) insertData(encSubj, encPred, encObj, encGraph string) {
	if ! i.inDataStatement {
		i.statements += "INSERT DATA {\n"
		i.inDataStatement = true
	}
	if i.currentGraph != encGraph {
		i.closeGraph("\n")
		i.statements += " GRAPH " + encGraph + " {\n"
		i.currentGraph = encGraph
	}
	if i.currentSubject != encSubj {
		i.closeSubject("\n")
		i.statements += "  " + encSubj + " "
		i.currentSubject = encSubj
	}
	if i.currentPredicate != encPred {
		i.closePredicate("\n   ")
		i.statements += encPred + " "
		i.currentPredicate = encPred
	}
	i.closeObject("\n    ")
	i.statements += encObj
	i.inObject = true
}

func (i *inserter) terminate() string {
	i.closeData("")
	return i.statements
}
