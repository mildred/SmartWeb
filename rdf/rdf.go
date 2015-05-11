package rdf

import (
	"errors"
	"fmt"
	"github.com/mildred/golibrdf"
	"log"
	"net/url"
	"os"
	"path/filepath"
)

type RedlandDataSet struct {
	World   *golibrdf.World
	Storage *golibrdf.Storage
	Model   *golibrdf.Model
}

func logMessage(message string) int {
	log.Println(message)
	return 1
}

func CreateRedlandDataSet(directory string) (*RedlandDataSet, error) {
	w := golibrdf.NewWorld()
	err := w.Open()
	if err != nil {
		return nil, err
	}

	//w.SetError(logMessage)
	//w.SetWarning(logMessage)

	storage_opts, err := golibrdf.NewHash(w)
	if err != nil {
		w.Close()
		return nil, err
	}

	sqlite_file := filepath.Join(directory, "redland.sqlite")

	if st, e := os.Stat(directory); e == nil {
		if !st.IsDir() {
			storage_opts.Free()
			w.Close()
			return nil, fmt.Errorf("%s: Must be a directory", directory)
		}
		if _, e := os.Stat(sqlite_file); e != nil {
			storage_opts.PutStrings("new", "yes")
			log.Printf("Create RDF storage: %s\n", sqlite_file);
		}
	} else {
		err := os.MkdirAll(directory, 0777)
		if err != nil {
			storage_opts.Free()
			w.Close()
			return nil, err
		}
		storage_opts.PutStrings("new", "yes")
		log.Printf("Create RDF storage: %s\n", sqlite_file);
	}

	/*
		//storage_opts.PutStrings("hash-type", "memory")
		storage_opts.PutStrings("hash-type", "bdb")
		storage_opts.PutStrings("contexts", "yes")
		//storage_opts.PutStrings("dir", ".")
		storage_opts.PutStrings("dir", directory)

		storage, err := golibrdf.NewStorageWithOptions(w, "hashes", "rdf", storage_opts)
		//storage, err := golibrdf.NewStorage(w, "hashes", "db1", "hash-type='memory'")
		storage_opts.Free()
		if err != nil {
			storage_opts.Free()
			w.Close()
			return nil, err
		}
	*/

	storage, err := golibrdf.NewStorageWithOptions(w, "sqlite", sqlite_file, storage_opts)
	storage_opts.Free()
	if err != nil {
		storage_opts.Free()
		w.Close()
		return nil, err
	}

	model, err := golibrdf.NewModel(w, storage, "")
	if err != nil {
		storage.Free()
		w.Close()
	}

	return &RedlandDataSet{
		World:   w,
		Storage: storage,
		Model:   model,
	}, nil
}

func (ds *RedlandDataSet) makeNode(o interface{}) (*golibrdf.Node, error) {
	if u, ok := o.(*url.URL); ok {
		return golibrdf.NewNodeFromUriString(ds.World, u.String())
	} else if s, ok := o.(string); ok {
		return golibrdf.NewNodeFromLiteral(ds.World, s)
	} else {
		return nil, fmt.Errorf("Could not make a node from %#v", o)
	}
}

func (ds *RedlandDataSet) AddQuad(context, subject, predicate, object interface{}) error {
	nContext, err := ds.makeNode(context)
	if err != nil {
		return err
	}

	nSubject, err := ds.makeNode(subject)
	if err != nil {
		nContext.Free()
		return err
	}

	nPredicate, err := ds.makeNode(predicate)
	if err != nil {
		nSubject.Free()
		nContext.Free()
		return err
	}

	nObject, err := ds.makeNode(object)
	if err != nil {
		nPredicate.Free()
		nSubject.Free()
		nContext.Free()
		return err
	}

	statement, err := golibrdf.NewStatementFromNodes(ds.World, nSubject, nPredicate, nObject)
	if err != nil {
		nObject.Free()
		nPredicate.Free()
		nSubject.Free()
		nContext.Free()
		return err
	}

	err = ds.Model.ContextAddStatement(nContext, statement)
	if err != nil {
		statement.Free()
		nContext.Free()
		return err
	}

	return nil
}

func (ds *RedlandDataSet) Close() {
	ds.Model.Free()
	ds.Storage.Free()
	ds.World.Close()
}

func (ds *RedlandDataSet) QueryGraph(query, content_type string) ([]byte, error) {
	q, err := golibrdf.NewQuery(ds.World, "sparql", query)
	if err != nil {
		return []byte{}, err
	}

	res, err := ds.Model.Execute(&q)
	if err != nil {
		return []byte{}, err
	}
	defer res.Free()

	stream := res.AsStream()
	if stream == nil {
		return []byte{}, errors.New("Result is probably not a graph")
	}
	defer stream.Free()

	storage, err := golibrdf.NewStorage(ds.World, "hashes", "results", "hash-type='memory'")
	if err != nil {
		return []byte{}, err
	}
	defer storage.Free()

	model, err := golibrdf.NewModel(ds.World, storage, "")
	if err != nil {
		return []byte{}, err
	}
	defer model.Free()

	err = model.AddStatements(stream)
	if err != nil {
		return []byte{}, err
	}

	serializer, err := golibrdf.NewSerializer(ds.World, "", content_type, nil)
	if err != nil {
		return []byte{}, err
	}
	defer serializer.Free()

	resultString, err := serializer.SerializeModelToString(model, nil)
	if err != nil {
		return []byte{}, err
	}

	return []byte(resultString), nil
}
