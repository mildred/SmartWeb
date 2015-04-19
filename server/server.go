package server

import (
	"net/http"
	"filepath"
)

type SmartServer struct {
	DataPath string
}

func (server SmartServer) ServeHTTP(http.ResponseWriter, req *http.Request) {
	var p := filepath.Join(server.DataPath, filepath.Clean(req.URL.Path))

	
}
