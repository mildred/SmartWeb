package server

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type SmartServer struct {
	DataPath string
}

func handleError(res http.ResponseWriter, status int, err string) {
	res.Header().Set("Content-Type", "text/plain, charset=utf-8")
	res.WriteHeader(http.StatusNotFound)
	res.Write([]byte(err))
}

func (server SmartServer) getFile(path string) string {
	var p = filepath.Clean(path)
	if p[0] == '/' {
		p = p[1:]
	}
	p = strings.Replace(p, "/", ".dir/", -1)
	p = filepath.Join(server.DataPath, p)
	return p
}

func (server SmartServer) handleGET(res http.ResponseWriter, req *http.Request) {
	var p = server.getFile(req.URL.Path)

	var meta MetaData = read(p)
	for header := range meta.Headers {
		res.Header().Set(header, meta.Headers[header])
	}

	var f, err = os.Open(p + ".data")
	if err != nil {
		handleError(res, http.StatusNotFound, err.Error())
		return
	}

	res.WriteHeader(http.StatusOK)

	if req.Method != "HEAD" {
		io.Copy(res, f)
	}

	f.Close()
}

func (server SmartServer) handlePUT(res http.ResponseWriter, req *http.Request) {
	var p = server.getFile(req.URL.Path)
	var dir = filepath.Dir(p)
	os.MkdirAll(dir, 0777)

	var meta MetaData = createMeta()
	if contentType := req.Header.Get("Content-Type"); contentType != "" {
		meta.Headers["Content-Type"] = contentType
	}

	err := meta.write(p)
	if err != nil {
		handleError(res, http.StatusInternalServerError, err.Error())
		return
	}

	var successStatus = http.StatusNoContent
	if _, e := os.Stat(p + ".data"); e != nil {
		successStatus = http.StatusCreated
	}

	f, err := os.Create(p + ".data.tmp")
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

	err = os.Rename(p+".data.tmp", p+".data")
	if err != nil {
		handleError(res, http.StatusInternalServerError, err.Error())
		return
	}

	res.WriteHeader(successStatus)
}

func (server SmartServer) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	if req.Method == "GET" || req.Method == "HEAD" {
		server.handleGET(res, req)
	} else if req.Method == "PUT" {
		server.handlePUT(res, req)
	} else {
		res.WriteHeader(http.StatusMethodNotAllowed)
	}
}
