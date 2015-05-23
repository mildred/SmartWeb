package server2

import (
	"archive/zip"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"log"
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
	
	z, err := zip.NewReader(f, req.ContentLength)
	if err != nil {
		handleError(res, 500, err.Error())
		return
	}

    for _, f := range z.File {
		log.Println(f.Name)
	}
}
