package main

import (
	"flag"
	"github.com/mildred/SmartWeb/bundle"
	"log"
	"os"
	"path/filepath"
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
		
		// FIXME use base URI provided in args and content type
		return b.InsertFile(path, path, f)

	}
}
