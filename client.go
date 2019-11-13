package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"
)

func main() {
	// Parse Arguments
	var id int
	var configPath string
	flag.IntVar(&id, "id", 0, "Which Client this is")
	flag.StringVar(&configPath, "config", "config.json", "Where the config file is located")

	flag.Parse()

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

	var config Config
	err = json.Unmarshal(b, &config)
	if err != nil {
		log.Fatal(err)
	}

	myAddr := config.Clients[id].Address
	myPort := config.Clients[id].Port
	fmt.Println("ME")
	fmt.Println(myAddr)
	fmt.Println(myPort)

	go broadcaster()
	go writeMessages()

	// Contact Everyone Else
	for i, client := range config.Clients {
		if i == id {
			continue
		}
		// Attempt to Dial
		fmt.Println("Ringing ", client.Port)
		conn, err := net.Dial(
			"tcp", fmt.Sprintf("%s:%d", client.Address, client.Port),
		)
		// If this worked use this as our communication chanel
		if err == nil {
			fmt.Println("Picked Up!")
			go handleConn(conn)
		}

	}

	// Start a listening server
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", myAddr, myPort))
	if err != nil {
		log.Fatal(err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Print(err)
			continue
		}
		fmt.Println("Call Received!")
		go handleConn(conn)
	}
}

type Config struct {
	Clients []struct {
		Address string
		Port    int
	}
}

func writeMessages() {
	input := bufio.NewScanner(os.Stdin)
	for input.Scan() {
		messages <- input.Text()
	}
	log.Print("Write Messages is Donezo")
}

func handleConn(conn net.Conn) {
	ch := make(chan string) // outgoing client messages
	go clientWriter(conn, ch)

	entering <- ch

	input := bufio.NewScanner(conn)
	for input.Scan() {
		fmt.Println(input.Text())
	}
	fmt.Println("Done: ", strings.Split(conn.LocalAddr().String(), ":")[1])
	leaving <- ch
	conn.Close()
}

func clientWriter(conn net.Conn, ch <-chan string) {
	for msg := range ch {
		fmt.Fprintln(conn, msg) // NOTE: ignoring network errors
	}
}

// BroadCaster
type client chan<- string // an outgoing message channel

var (
	entering = make(chan client)
	leaving  = make(chan client)
	messages = make(chan string) // all incoming client messages
)

func broadcaster() {
	clients := make(map[client]bool) // all connected clients
	for {
		select {
		case msg := <-messages:
			// Broadcast incoming message to all
			// clients' outgoing message channels.
			for cli := range clients {
				cli <- msg
			}

		case cli := <-entering:
			clients[cli] = true

		case cli := <-leaving:
			delete(clients, cli)
			close(cli)
		}
	}
}
