package main

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	// Parse Arguments
	var id int
	var configPath string
	flag.IntVar(&id, "id", 0, "Which Client this is")
	flag.StringVar(&configPath, "config", "config.json", "Where the config file is located")

	flag.Parse()

	// Read and Parse Config File
	ReadConfig(configPath, id)

	// Start Reading Messages
	go readMessages()

	// Start Pooling Connections
	StartConnectionManager()

	// Start Broadcasting
	finished := make(chan bool)
	go writeMessages(finished)
	// Wait for EOF before shutting down
	<-finished
	log.Println("Main Thread Exiting")
}

// This go routine writes messages to the broadcaster
func writeMessages(finished chan bool) {
	input := bufio.NewScanner(os.Stdin)
	for input.Scan() {
		cm := ClientMessage{Message: input.Text()}
		Messages <- DataMessage{Message: ClientMessageToBytes(cm)}
	}
	log.Println("Shutting Off")
	finished <- true
}

// This go routine reads incoming messages and prints them
func readMessages() {
	for incomingMessage := range IncomingMessages {
		clientMessage := BytesToClientMessage(incomingMessage.Data.Message)
		fmt.Printf("%s: %s\n", incomingMessage.Client, clientMessage.Message)
	}
}

type ClientMessage struct {
	Message string
}

func BytesToClientMessage(input []byte) ClientMessage {
	var cm ClientMessage
	dec := gob.NewDecoder(bytes.NewReader(input))
	err := dec.Decode(&cm)
	if err != nil {
		log.Printf("Error Decoding: %s", err)
	}
	return cm
}

func ClientMessageToBytes(input ClientMessage) []byte {
	encBuf := new(bytes.Buffer)
	err := gob.NewEncoder(encBuf).Encode(input)
	if err != nil {
		log.Printf("Error Encoding: %s", err)
	}
	return encBuf.Bytes()
}
