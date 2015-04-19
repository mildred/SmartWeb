package server

import (
	"io"
	"os"
	"path"
)

type Entry interface {
	Name() string
	Exists() bool

	Create() (EntryWriter, error)
	Open() (EntryReader, error)
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
	Path string
}

type FSEntryWriter struct {
	os.File
}

func (e FSEntry) Name() string {
	return path.Base(e.Path)
}

func (e FSEntry) Exists() bool {
	_, err := os.Stat(e.Path + ".data")
	return err != nil
}

func (e FSEntry) Create() (EntryWriter, error) {
	f, err := os.Create(e.Path + ".data.tmp")
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
	return os.Open(e.Path + ".data")
}

func (e FSEntry) Delete() error {
	return os.Remove(e.Path + ".data")
}

func (e FSEntry) DeleteAll() error {
	err := os.Remove(e.Path + ".data")
	if err != nil {
		return err
	}

	err = os.RemoveAll(e.Path + ".meta")
	if err != nil {
		return err
	}

	err = os.RemoveAll(e.Path + ".dir")
	return err
}

func (e FSEntry) Parameters() Entry {
	return FSEntry{
		Path: e.Path + ".meta",
	}
}

func (e FSEntry) Children() ([]Entry, error) {
	f, err := os.Open(e.Path + ".dir")
	if err != nil {
		return nil, err
	}

	names, err := f.Readdirnames(-1)

	var res []Entry

	for i := range names {
		res = append(res, FSEntry{Path: path.Join(e.Path+".dir", names[i])})
	}

	return res, err
}

func (e FSEntry) Child(name string) Entry {
	return FSEntry{
		Path: path.Join(e.Path+".dir", name),
	}
}
