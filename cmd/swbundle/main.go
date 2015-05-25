package main

import (
	"flag"
	"github.com/mildred/SmartWeb/bundle"
	"github.com/mildred/SmartWeb/nquads"
	"log"
	"os"
	"path/filepath"
	"net/http"
	"io"
)

func main() {
	// FIXME support index file and insert it in graph
	// FIXME add sw:child
	// FIXME detect content type
	baseUri := flag.String("base", "", "Base URI")
	flag.Parse()
	bundleFile := flag.Arg(0)
	source := flag.Arg(1)
	
	var err error
	if source != "" {
		err = writeBundle(bundleFile, source, *baseUri)
	} else {
		err = readBundle(bundleFile)
	}

	if err != nil {
		log.Fatalln(err)
	}
}

func writeBundle(bundleFile, sourceDir, baseUri string) error {
	f, err := os.Create(bundleFile)
	if err != nil {
		return err
	}
	defer f.Close()
	
	b, err := bundle.NewWriter(f, baseUri)
	if err != nil {
		return err
	}

	defer b.Close()
	
	err = readDir(b, sourceDir, "", nil)
	if err != nil {
		return err
	}
	return nil
}

func readBundle(bundleFile string) error {
	r, err := bundle.OpenReader(bundleFile)
	if err != nil {
		return err
	}

	defer r.Close()
	
	for value := range r.GraphStatements(64) {
		switch st := value.(type) {
			case error: return st
			case *nquads.Statement: log.Printf("%s\n", st.String())
			default: panic(value)
		}
	}

	return nil
}

func readDir(b *bundle.Writer, prefix, path string, parentpath *string) error {
	f, err := os.Open(filepath.Join(prefix, path))
	if err != nil {
		return err
	}
	
	defer f.Close()

	names, err := f.Readdirnames(-1)
	if err == nil {
		if path != "" {
			path = path + "/"
		}
		
		b.WriteTriple(
			path,
			"tag:mildred.fr,2015-05:SmartWeb#relativePath",
			path)

		if parentpath != nil {
			err = b.WriteQuadIri(
				*parentpath,
				"tag:mildred.fr,2015-05:SmartWeb#child",
				path,
				*parentpath)
			if err != nil {
				return err
			}
		}

		for _, name := range names {
			p := filepath.Join(path, name)
			
			err = readDir(b, prefix, p, &path)
			if err != nil {
				return err
			}
		}
		return nil

	} else {

		if parentpath != nil {
			err = b.WriteQuadIri(
				*parentpath,
				"tag:mildred.fr,2015-05:SmartWeb#child",
				path,
				*parentpath)
			if err != nil {
				return err
			}
		}

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
