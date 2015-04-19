package server

import (
	"encoding/json"
	"log"
	"os"
)

type MetaData struct {
	Headers map[string]string
}

func createMeta() MetaData {
	return MetaData{
		Headers: make(map[string]string),
	}
}

func read(p string) MetaData {
	var metadata MetaData

	f, err := os.Open(p + ".meta.json")
	if err != nil {
		log.Println(err)
		return metadata
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	err = dec.Decode(&metadata)
	if err != nil {
		log.Println(err)
	}
	return metadata
}

func (meta MetaData) write(p string) error {
	f, err := os.Create(p + ".meta.json.tmp")
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	err = enc.Encode(meta)
	if err != nil {
		return err
	}

	err = os.Rename(p+".meta.json.tmp", p+".meta.json")
	if err != nil {
		return err
	}

	return nil
}
