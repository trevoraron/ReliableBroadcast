package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
)

type Config struct {
	Clients []struct {
		Address string
		Port    int
	}
}

var GlobalConfig Config
var ID int

func ReadConfig(configPath string, id int) {
	// Write ID
	ID = id

	// Open Config File and Read it
	file, err := os.Open(configPath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	b, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal(b, &GlobalConfig)
	if err != nil {
		log.Fatal(err)
	}
}
