package bundle

import (
	"archive/zip"
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"github.com/mildred/SmartWeb/nquads"
	"io"
	"strings"
)

type Writer struct {
	*zip.Writer
	nquads.NQuadWriter
	Graphs *bytes.Buffer
}

func NewWriter(f io.Writer, baseUri string) (*Writer, error) {
	bytesBuffer := &bytes.Buffer{}
	w := &Writer{ zip.NewWriter(f), nquads.NQuadWriter{bytesBuffer}, bytesBuffer }

	mimetype, err := w.Writer.CreateHeader(&zip.FileHeader{
		Name:   "mimetype",
		Method: zip.Store,
	})
	if err != nil {
		return nil, err
	}

	_, err = mimetype.Write([]byte("application/smartweb-bundle+zip"))
	if err != nil {
		return nil, err
	}
	
	w.WriteComment(" Relocatable SmartWeb Graph")
	w.WriteTripleIri(
		"",
		"http://www.w3.org/1999/02/22-rdf-syntax-ns#type",
		"tag:mildred.fr,2015-05:SmartWeb#RelocatableGraph")
	if baseUri != "" {
		w.WriteTripleIri(
			"",
			"tag:mildred.fr,2015-05:SmartWeb#baseUri",
			baseUri)
	}

	return w, nil
}

func (w *Writer) InsertFile(fullUri, name string, f io.ReadSeeker) error {
	_, err := f.Seek(0, 0)
	if err != nil {
		return err
	}

	h := sha1.New()

	_, err = io.Copy(h, f)
	if err != nil {
		return err
	}

	_, err = f.Seek(0, 0)
	if err != nil {
		return err
	}

	sha1name := fmt.Sprintf("sha1:%s", strings.ToLower(hex.EncodeToString(h.Sum([]byte{}))))

	datafile, err := w.Writer.Create(sha1name)
	if err != nil {
		return err
	}

	_, err = io.Copy(datafile, f)
	if err != nil {
		return err
	}
	
	w.WriteEmptyLine()
	w.WriteComment(" " + name)
	
	w.WriteTriple(
		fullUri,
		"tag:mildred.fr,2015-05:SmartWeb#relativePath",
		name)

	w.WriteQuadIri(
		fullUri,
		"tag:mildred.fr,2015-05:SmartWeb#hash",
		sha1name,
		fullUri)

	return nil
}

func (w *Writer) Close() error {
	zgraphs, err := w.Writer.Create("graphs.nq")
	if err != nil {
		return err
	}

	_, err = zgraphs.Write(w.Graphs.Bytes())
	if err != nil {
		return err
	}

	w.Writer.Close()
	return nil
}
