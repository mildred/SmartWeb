package main

import (
	"flag"
	"github.com/mildred/SmartWeb/bundle"
	"log"
	"os"
	"path/filepath"
	"net/http"
	"io"
)

func main() {
	baseUri := flag.String("base", "", "Base URI")
	flag.Parse()
	bundleFile := flag.Arg(0)
	source := flag.Arg(1)

	f, err := os.Create(bundleFile)
	if err != nil {
		log.Fatalln(err)
		return
	}
	defer f.Close()
	
	b, err := bundle.NewWriter(f, *baseUri)
	if err != nil {
		log.Fatalln(err)
		return
	}

	defer b.Close();
	
	err = readDir(b, source, "")
	if err != nil {
		log.Fatalln(err)
		return
	}
}

func readDir(b *bundle.Writer, prefix, path string) error {
	f, err := os.Open(filepath.Join(prefix, path))
	if err != nil {
		return err
	}

	names, err := f.Readdirnames(-1)
	if err == nil {

		for _, name := range names {
			p := filepath.Join(path, name)
			err = readDir(b, prefix, p)
			if err != nil {
				return err
			}
		}
		return nil

	} else {

		var firstBytes [512]byte
		io.ReadFull(f, firstBytes[:])
		mimeType := http.DetectContentType(firstBytes[:])
		
		// FIXME use base URI provided in args
		// FIXME detect more text/* content types based on extension
		fullUri := path
		err := b.InsertFile(fullUri, path, f)
		if err != nil {
			return err
		}

		return b.WriteQuad(
			fullUri,
			"tag:mildred.fr,2015-05:SmartWeb#contentType",
			mimeType,
			fullUri)

	}
}
