package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

func main() {
	var id int
	var configPath string
	flag.IntVar(&id, "id", 0, "Which Client this is")
	flag.StringVar(&configPath, "config", "config.json", "Where the config file is located")

	flag.Parse()

	file, err := os.Open(configPath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	b, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}

	var config Config
	err = json.Unmarshal(b, &config)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(config.Clients[id].Address)
	fmt.Println(config.Clients[id].Port)
}

type Config struct {
	Clients []struct {
		Address string
		Port    int
	}
}
