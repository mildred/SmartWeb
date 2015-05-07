package rdf

import (
	"github.com/mildred/golibrdf"
	"os"
	"net/url"
	"fmt"
)

type RedlandDataSet struct {
	World   *golibrdf.World
	Storage *golibrdf.Storage
	Model   *golibrdf.Model
}

func CreateRedlandDataSet(directory string) (*RedlandDataSet, error) {
	w := golibrdf.NewWorld()
	err := w.Open()
	if err != nil {
		return nil, err
	}
	
	storage_opts, err := golibrdf.NewHash(w);
	if err != nil {
		w.Close();
		return nil, err
	}
	
	if st, e := os.Stat(directory); e == nil {
		if !st.IsDir() {
			storage_opts.Free();
			w.Close();
			return nil, fmt.Errorf("%s: Must be a directory", directory);
		}
	} else {
		err := os.MkdirAll(directory, 0777)
		if err != nil {
			storage_opts.Free();
			w.Close();
			return nil, err
		}
		storage_opts.PutStrings("new", "yes");
	}
	
	storage_opts.PutStrings("hash-type", "bdb");
	storage_opts.PutStrings("contexts", "yes");
	storage_opts.PutStrings("dir", directory);
	
	storage, err := golibrdf.NewStorageWithOptions(w, "hashes", directory, storage_opts)
	storage_opts.Free()
	if err != nil {
		storage_opts.Free();
		w.Close();
		return nil, err
	}
	
	model, err := golibrdf.NewModel(w, storage, "")
	if err != nil {
		storage.Free();
		w.Close();
	}

	return &RedlandDataSet{
		World: w,
		Storage: storage,
		Model: model,
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
		return err;
	}
	
	return nil
}

func (ds *RedlandDataSet) Close() {
	ds.Model.Free()
	ds.Storage.Free()
	ds.World.Close();
}
