package server

import (
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

type Entry interface {
	Name() string
	Exists() bool

	Create() (EntryWriter, error)
	Open() (EntryReader, error)
	Read() ([]byte, error)
	Write([]byte) error
	Delete() error
	DeleteAll() error

	Parameters() Entry
	Children() ([]Entry, error)
	Child(pathname string) Entry
}

type EntryReader interface {
	io.Reader
	io.Seeker
	io.Closer
}

type EntryWriter interface {
	io.Writer
	io.Closer
	Commit() error
}

type FSEntry struct {
	PathDot string
	name    string
}

type FSEntryWriter struct {
	os.File
}

func (e FSEntry) Name() string {
	return e.name
}

func (e FSEntry) Exists() bool {
	_, err := os.Stat(e.PathDot + "data")
	return err != nil
}

func (e FSEntry) Create() (EntryWriter, error) {
	os.MkdirAll(path.Dir(e.PathDot), 0777)

	f, err := os.Create(e.PathDot + "data-tmp")
	if f == nil {
		return nil, err
	} else {
		return &FSEntryWriter{*f}, err
	}
}

func (f *FSEntryWriter) Commit() error {
	old := f.Name()
	new := old[:len(old)-4]
	return os.Rename(old, new)
}

func (e FSEntry) Open() (EntryReader, error) {
	return os.Open(e.PathDot + "data")
}

func (e FSEntry) Read() ([]byte, error) {
	f, err := e.Open()
	if err != nil {
		return []byte{}, err
	}
	defer f.Close()

	return ioutil.ReadAll(f)
}

func (e FSEntry) Write(s []byte) error {
	f, err := e.Create()
	if err != nil {
		return err
	}
	defer f.Commit()
	defer f.Close()

	_, err = f.Write(s)
	return err
}

func (e FSEntry) Delete() error {
	err := os.Remove(e.PathDot + "data")
	if err != nil && os.IsNotExist(err) {
		return nil
	}
	return err
}

func (e FSEntry) DeleteAll() error {
	err := os.Remove(e.PathDot + "data")
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	err = os.RemoveAll(e.PathDot + "meta")
	if err != nil {
		return err
	}

	err = os.RemoveAll(e.PathDot + "metadir")
	if err != nil {
		return err
	}

	err = os.RemoveAll(e.PathDot + "dir")
	return err
}

func (e FSEntry) Parameters() Entry {
	return &FSEntry{
		name:    "",
		PathDot: e.PathDot + "meta",
	}
}

func (e FSEntry) Children() ([]Entry, error) {
	f, err := os.Open(e.PathDot + "dir")
	if err != nil {
		return nil, err
	}

	names, err := f.Readdirnames(-1)

	var res []Entry

	for i := range names {
		name := names[i]
		e := &FSEntry{
			name:    name,
			PathDot: path.Join(e.PathDot+"dir", name) + ".",
		}
		res = append(res, e)
	}

	return res, err
}

func (e FSEntry) Child(name string) Entry {
	n := name
	for len(n) > 0 && n[0] == '/' {
		n = n[1:]
	}
	if len(n) == 0 {
		return e
	}

	n = strings.Replace(n, "/", ".dir/", -1)
	p := path.Join(e.PathDot+"dir", n)
	if name[len(name)-1] == '/' {
		p = p + "/"
	} else {
		p = p + "."
	}

	return &FSEntry{
		name:    path.Base(n),
		PathDot: p,
	}
}
