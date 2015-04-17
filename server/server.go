package server

import (
	"net/http"
)

type SmartServer struct {
	DataPath string
}

func (server SmartServer) ServeHTTP(http.ResponseWriter, *http.Request) {

}
