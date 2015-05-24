package server2

import (
	"crypto"
	"crypto/sha1"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"github.com/mildred/SmartWeb/sparql"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var SmartWeb_hasReferer, _ = url.Parse("tag:mildred.fr,2015-05:SmartWeb#hasReferer")

type SmartServer struct {
	Root        string
	Certificate *x509.Certificate
	PrivateKey  crypto.PrivateKey
	dataSet     *sparql.Client
	useAcl      bool
}

func CreateFileServer(path string, Certificate *x509.Certificate, PrivateKey crypto.PrivateKey, query, update string, useAcl bool) *SmartServer {
	return &SmartServer{
		Root:        path,
		Certificate: Certificate,
		PrivateKey:  PrivateKey,
		dataSet:     sparql.NewClient(query, update),
		useAcl:      useAcl,
	}
}

func handleError(res http.ResponseWriter, status int, err string) {
	res.Header().Set("Content-Type", "text/plain, charset=utf-8")
	res.WriteHeader(status)
	res.Write([]byte(err))
}

func (server SmartServer) handleGET(u *url.URL, res http.ResponseWriter, req *http.Request) {
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
		handleError(res, 404, "")
		return
	}

	binding := result.Results.Bindings[0]
	hash := binding["hash"]
	content_type := binding["type"]

	f, err := os.Open(filepath.Join(server.Root, hash.Value))
	if err != nil {
		handleError(res, 404, err.Error())
		return
	}

	res.Header().Set("Content-Type", content_type.Value)
	res.Header().Set("Etag", hash.Value)
	res.WriteHeader(http.StatusOK)

	if req.Method != "HEAD" {
		io.Copy(res, f)
	}
}

func (server SmartServer) handlePUT(u *url.URL, res http.ResponseWriter, req *http.Request) {

	f, err := ioutil.TempFile(server.Root, "temp:")
	if err != nil {
		handleError(res, 500, err.Error())
		return
	}
	defer f.Close()

	hash := sha1.New()
	_, err = io.Copy(f, io.TeeReader(req.Body, hash))
	if err != nil {
		handleError(res, 500, err.Error())
		return
	}

	digest := hex.EncodeToString(hash.Sum([]byte{}))
	uri := fmt.Sprintf("sha1:%s", strings.ToLower(digest))

	res.Header().Set("Hash", uri)

	err = os.Rename(f.Name(), filepath.Join(server.Root, uri))
	if err != nil {
		go os.Remove(f.Name())
		handleError(res, 500, err.Error())
		return
	}

	var parentChain string
	urls := urlParents(u)
	for i := len(urls) - 1; i > 0; i-- {
		parentChain += sparql.MakeQuery(" %2u sw:child %1u .", &urls[i-1], &urls[i])
	}

	_, err = server.dataSet.Update(sparql.MakeQuery(`
		PREFIX sw: <tag:mildred.fr,2015-05:SmartWeb#>
		
		CLEAR SILENT GRAPH %1u;
		INSERT DATA {
			GRAPH %1u {
				%1u
					sw:hash        %2u ;
					sw:contentType %3s .
				%4q
			}
		}
	`, u, uri, req.Header.Get("Content-Type"), parentChain))

	if err != nil {
		handleError(res, 500, err.Error())
		return
	}

	res.WriteHeader(http.StatusCreated)
}

func urlParents(u *url.URL) []url.URL {
	var parentUrl url.URL = *u
	var parentChain []url.URL = []url.URL{parentUrl}
	if parentUrl.Fragment != "" {
		parentUrl.Fragment = ""
		parentChain = append(parentChain, parentUrl)
	}
	if parentUrl.RawQuery != "" {
		parentUrl.RawQuery = ""
		parentChain = append(parentChain, parentUrl)
	}
	for parentUrl.Path != "/" {
		// Remove trailing / from URL
		if parentUrl.Path[len(parentUrl.Path)-1] == '/' {
			parentUrl.Path = parentUrl.Path[:len(parentUrl.Path)-1]
		}
		// Get dirname from URL
		parentUrl.Path = filepath.Dir(parentUrl.Path)
		// Add trailing / to directory
		if parentUrl.Path != "/" {
			parentUrl.Path = parentUrl.Path + "/"
		}
		parentChain = append(parentChain, parentUrl)
	}
	return parentChain
}

func (server SmartServer) handleDELETE(u *url.URL, res http.ResponseWriter, req *http.Request) {
	result, err := server.dataSet.Select(sparql.MakeQuery(`
		PREFIX sw: <tag:mildred.fr,2015-05:SmartWeb#>
		SELECT ?hash
		WHERE { %1u sw:hash ?hash . }
	`, u))
	if err != nil {
		handleError(res, 500, err.Error())
		return
	}

	if len(result.Results.Bindings) < 1 {
		handleError(res, 404, "Not Found")
		return
	}

	hash := result.Results.Bindings[0]["hash"].Value
	_, err = server.dataSet.Update(sparql.MakeQuery(`
		PREFIX sw: <tag:mildred.fr,2015-05:SmartWeb#>
		DROP SILENT GRAPH %1u
	`, u))
	if err != nil {
		handleError(res, 500, err.Error())
		return
	}

	result, err = server.dataSet.Select(sparql.MakeQuery(`
		PREFIX sw: <tag:mildred.fr,2015-05:SmartWeb#>
		SELECT (count(?subj) AS ?count)
		WHERE { ?subj sw:hash %1u . }
	`, hash))
	if err != nil {
		handleError(res, 500, err.Error())
		return
	}

	count, err := strconv.ParseInt(result.Results.Bindings[0]["count"].Value, 10, 0)
	if err == nil && count == 0 {
		err := os.Remove(filepath.Join(server.Root, hash))
		if err != nil {
			log.Println(err)
		}
	}
}

func (server SmartServer) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	curUrl := (&url.URL{
		Scheme: "http", // Ignore https to avoid breaking links
		Host:   req.Host,
	}).ResolveReference(req.URL)

	log.Println(req.Method + " " + curUrl.String())

	if req.URL.RawQuery == "keygen" {
		server.handleKeygen(res, req)
		return
	}
	
	if server.useAcl {
		auth := false
		if req.TLS != nil {
			for _, clientCert := range req.TLS.PeerCertificates {
				var err error
				userid := fmt.Sprintf("x509-certificate-fingerprint:sha256:%s", strings.ToLower(hex.EncodeToString(SHA256Fingerprint(*clientCert))))
				auth, err = checkAuth(server.dataSet, curUrl, req.Method, userid)
				if err != nil {
					handleError(res, 500, err.Error())
					return
				}
				if auth {
					break
				}
			}
		} else {
			var err error
			auth, err = checkAuth(server.dataSet, curUrl, req.Method, "tag:mildred.fr,2015-05:SmartWeb#Anonymous")
			if err != nil {
				handleError(res, 500, err.Error())
				return
			}
		}
	
		if !auth {
			// RFC 6797: HTTP Strict Transport Security (HSTS)
			res.Header().Set("Strict-Transport-Security", "max-age=1")
			handleError(res, 403, "Unauthorized")
			return
		}
	}

	go func() {
		referrer, err := url.Parse(req.Referer())
		if err != nil {
			return
		}
		referrer = curUrl.ResolveReference(referrer)
		// Add the referrer in storage
		err = server.dataSet.AddQuad(curUrl, curUrl, SmartWeb_hasReferer, referrer)
		if err != nil {
			log.Println(err)
		}
	}()

	if req.Method == "GET" || req.Method == "HEAD" {
		server.handleGET(curUrl, res, req)
	} else if req.Method == "PUT" {
		server.handlePUT(curUrl, res, req)
	} else if req.Method == "POST" {
		server.handlePOSTBundle(curUrl, res, req)
	} else if req.Method == "DELETE" {
		server.handleDELETE(curUrl, res, req)
	} else {
		res.WriteHeader(http.StatusMethodNotAllowed)
	}
}

var header_value_regexp, _ = regexp.Compile(`([^,"]*|"([^"\\]*|\\.)*")*`)

func splitHeader(header string) []string {
	var result []string
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
