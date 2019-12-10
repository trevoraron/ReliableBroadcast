package main

import (
	"bufio"
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
		byteArr, err := StructToBytes(cm)
		if err != nil {
			log.Fatalf("Error Encoding Client Message: %s", err)
		}
		Messages <- DataMessage{Message: byteArr}
	}
	log.Println("Shutting Off")
	finished <- true
}

// This go routine reads incoming messages and prints them
func readMessages() {
	for incomingMessage := range IncomingMessages {
		var clientMessage ClientMessage
		err := BytesToStruct(incomingMessage.Data.Message, &clientMessage)
		if err != nil {
			log.Fatalf("Error Decoding Client Message: %s", err)
		}
		fmt.Printf("%s: %s\n", incomingMessage.Client, clientMessage.Message)
	}
}

type ClientMessage struct {
	Message string
}
