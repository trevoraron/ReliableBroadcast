package main

import (
	"bytes"
	"encoding/gob"
)

func BytesToStruct(input []byte, e interface{}) error {
	dec := gob.NewDecoder(bytes.NewReader(input))
	err := dec.Decode(e)
	if err != nil {
		return err
	}
	return nil
}

func StructToBytes(e interface{}) ([]byte, error) {
	encBuf := new(bytes.Buffer)
	err := gob.NewEncoder(encBuf).Encode(e)
	if err != nil {
		return nil, err
	}
	return encBuf.Bytes(), nil
}
