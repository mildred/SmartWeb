package server

import (
	"io"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strings"
)

// URI: /            Path: ~/data
//      /robots.txt        ~/robots.txt.data
//      /html              ~/html.data
//      /html/             ~/html.dir/data
//      /html/index.html   ~/html.dir/index.html.data
//      /file?meta=a       ~/file.metadir/a.data
//      /file?meta=/a      ~/file.metadir/a.data
//      /file?meta=/a/     ~/file.metadir/a.dir/data

type Entry interface {
	Name() string
	Exists() bool
	Parent(force bool) Entry

	Create() (EntryWriter, error)
	Open() (EntryReader, error)
	Read() ([]byte, error)
	Write([]byte) error
	Delete() error
	DeleteAll() error

	Parameters() Entry
	Children() ([]Entry, error)
	Child(pathname string) Entry
	Dir() Entry
	File() Entry
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
	level   int
}

type FSEntryWriter struct {
	os.File
}

func CreateFSEntry(p string) FSEntry {
	return FSEntry{
		PathDot: path.Clean(p) + "/",
		name:    "",
		level:   0,
	}
}

func (e FSEntry) Name() string {
	return e.name
}

func (e FSEntry) Exists() bool {
	_, err := os.Stat(e.PathDot + "data")
	return err != nil
}

func (e FSEntry) Parent(force bool) Entry {
	// Go to the entry file
	p := e.PathDot
	n := len(p)
	if n > 4 || p[n-4:] == "dir/" {
		p = p[:n-4]
	}

	i := strings.LastIndex(p, "/")
	if e.level == 0 || i < 0 {
		return nil
	}

	p = p[:i+1]

	j := strings.LastIndex(p[:i], ".")
	k := strings.LastIndex(p[:i], "/")

	if k >= j || j < 0 {
		return nil
	}

	return &FSEntry{
		PathDot: p,
		name:    p[k+1 : j],
		level:   e.level - 1,
	}
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
	return err
}

func (e FSEntry) Parameters() Entry {
	return &FSEntry{
		name:    "",
		PathDot: e.PathDot + "meta",
		level:   e.level + 1,
	}
}

func (e FSEntry) Children() ([]Entry, error) {
	// Go to the entry directory
	p := e.PathDot
	if p[len(p)-1] != '/' {
		p = p + "dir/"
	}

	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}

	names, err := f.Readdirnames(-1)
	sort.Strings(names)
	var res []Entry
	var lastName string

	for i, name := range names {
		j := strings.LastIndex(name, ".")
		if j == -1 {
			continue
		} else if i > 0 && lastName == name[:j] {
			continue
		}
		lastName = name[:j]
		e := &FSEntry{
			name:    lastName,
			PathDot: path.Join(p, lastName) + ".",
			level:   e.level + 1,
		}
		res = append(res, e)
	}

	return res, err
}

func (e FSEntry) Child(name string) Entry {
	var resName string
	var addlevel int

	// Go to the entry directory
	p := e.PathDot
	if p[len(p)-1] != '/' {
		p = p + "dir/"
	}

	// Go the children entries
	n := path.Clean("/" + name)[1:]
	if len(n) > 0 {
		elements := strings.Split(n, "/")
		p = path.Join(p, strings.Join(elements, ".dir/")) + "."
		resName = path.Base(n)
		addlevel = len(elements)
	} else {
		resName = e.name
		addlevel = 0
	}

	// If the last entry is a directory, go to the entry directory
	if len(name) > 0 && name[len(name)-1] == '/' && addlevel > 0 {
		p = p + "dir/"
	}

	return &FSEntry{
		name:    resName,
		level:   e.level + addlevel,
		PathDot: p,
	}
}

func (e FSEntry) Dir() Entry {
	// Go to the entry directory
	p := e.PathDot
	if p[len(p)-1] == '/' {
		return &e
	}

	return &FSEntry{
		name:    e.name,
		level:   e.level,
		PathDot: p + "dir/",
	}
}

func (e FSEntry) File() Entry {
	// Go to the entry file
	p := e.PathDot
	n := len(p)
	if n <= 4 || p[n-4:] != "dir/" {
		return &e
	}

	return &FSEntry{
		name:    e.name,
		level:   e.level,
		PathDot: p[:n-4],
	}
}
