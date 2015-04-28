package server

import (
	"io"
	"log"
	"net/http"
	"net/url"
)

type SmartServer struct {
	Root Entry
	auth Authenticator
}

func CreateFileServer(path string) *SmartServer {
	return &SmartServer{
		Root: CreateFSEntry(path),
		auth: CreateAuthenticator(),
	}
}

func handleError(res http.ResponseWriter, status int, err string) {
	res.Header().Set("Content-Type", "text/plain, charset=utf-8")
	res.WriteHeader(http.StatusNotFound)
	res.Write([]byte(err))
}

func (server SmartServer) getEntry(Url *url.URL) Entry {
	entry := server.Root.Child(Url.Path)
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

	if headers, err := meta.Child("headers").Children(); err == nil {
		for _, header := range headers {
			if data, err := header.Read(); err != nil {
				log.Println(err)
			} else {
				res.Header().Set(string(header.Name()), string(data))
			}
		}
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
	entry := server.getEntry(req.URL)

	if !server.auth.Authenticate(entry, res, req) {
		res.WriteHeader(http.StatusUnauthorized)
		return
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
