package main

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
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

	// Load Certs
	caCert, err := ioutil.ReadFile("./certs/ca.pem")
	if err != nil {
		log.Printf("failed to load cert: %s", err)
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	clientCertFile := fmt.Sprintf("./certs/client%d.pem", id)
	clientKeyFile := fmt.Sprintf("./certs/client%d-key.pem", id)
	clientCert, err := tls.LoadX509KeyPair(clientCertFile, clientKeyFile)

	serverCertFile := fmt.Sprintf("./certs/server%d.pem", id)
	serverKeyFile := fmt.Sprintf("./certs/server%d-key.pem", id)
	serverCert, err := tls.LoadX509KeyPair(serverCertFile, serverKeyFile)

	tlsConfigServer := &tls.Config{
		Certificates: []tls.Certificate{serverCert},  // server certificate which is validated by the client
		ClientCAs:    caCertPool,                     // used to verify the client cert is signed by the CA and is therefore valid
		ClientAuth:   tls.RequireAndVerifyClientCert, // this requires a valid client certificate to be supplied during handshake
	}

	tlsConfigClient := &tls.Config{
		Certificates: []tls.Certificate{clientCert}, // this certificate is used to sign the handshake
		RootCAs:      caCertPool,                    // this is used to validate the server certificate
	}
	tlsConfigClient.BuildNameToCertificate()

	go broadcaster()
	go writeMessages()

	// Contact Everyone Else
	for i, client := range config.Clients {
		if i == id {
			continue
		}
		// Attempt to Dial
		fmt.Println("Ringing ", client.Port)
		conn, err := tls.Dial(
			"tcp", fmt.Sprintf("%s:%d", client.Address, client.Port), tlsConfigClient,
		)
		// If this worked use this as our communication chanel
		if err == nil {
			fmt.Println("Picked Up!")
			go handleConn(conn)
		}
	}

	// Start a listening server
	listener, err := tls.Listen("tcp", fmt.Sprintf("%s:%d", myAddr, myPort), tlsConfigServer)

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

		tlsConn, ok := conn.(*tls.Conn)
		if !ok {
			log.Printf("failed to cast conn to tls.Conn")
			continue
		}

		go handleConn(tlsConn)
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

func handleConn(conn *tls.Conn) {
	defer conn.Close()

	tag := fmt.Sprintf("[%s -> %s]", conn.LocalAddr(), conn.RemoteAddr())
	log.Printf("%s accept", tag)

	err := conn.Handshake()
	if err != nil {
		log.Printf("failed to complete handshake: %s", err)
	}

	if len(conn.ConnectionState().PeerCertificates) > 0 {
		log.Printf("%s client common name: %+v", tag, conn.ConnectionState().PeerCertificates[0].Subject.CommonName)
	}

	ch := make(chan string) // outgoing client messages
	go clientWriter(conn, ch)

	entering <- ch

	input := bufio.NewScanner(conn)
	for input.Scan() {
		fmt.Println(input.Text())
	}
	fmt.Println("Done: ", strings.Split(conn.RemoteAddr().String(), ":")[1])
	leaving <- ch
}

func clientWriter(conn *tls.Conn, ch <-chan string) {
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
