package bundle

import (
	"io"
	"io/ioutil"
	"archive/zip"
	"github.com/mildred/SmartWeb/nquads"
)

type Reader struct {
	*zip.Reader
}

type ReadCloser struct {
	Reader
	io.Closer
}

func NewReader(f io.ReaderAt, size int64) (*Reader, error) {
	z, err := zip.NewReader(f, size)
	if err != nil {
		return nil, err
	}

	r := &Reader{ z }
	
	return r, nil
}

func OpenReader(name string) (*ReadCloser, error) {
	z, err := zip.OpenReader(name)
	if err != nil {
		return nil, err
	}

	r := &ReadCloser{ Reader{ &z.Reader }, z }
	
	return r, nil
}

func (r *Reader) Graph() (*nquads.ReadCloser, error) {
	for _, f := range r.Reader.File {
		if f.Name == "graphs.nq" {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			} else {
				return nquads.NewReadCloser(rc), nil
			}
		}
	}
	return nil, nil
}

func readGraph(r io.Reader) (string, error) {
	bytes, err := ioutil.ReadAll(r)
	return string(bytes), err
}
