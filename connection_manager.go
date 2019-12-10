package main

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
)

// Channel to a particular connection
type client chan<- string // an outgoing message channel

type IncomingMessage struct {
	Client  string
	Message string
}

// Global Channels
var (
	entering         = make(chan client)
	leaving          = make(chan client)
	Messages         = make(chan string) // all incoming client messages
	IncomingMessages = make(chan IncomingMessage)
)

// Sets everything up -- reads certs, starts the broadcaster channel, tries to dial other clients to
// generate mTLS connections, and then starts a listening server that will create connections when other
// clients ring this one
func StartConnectionManager() {
	myAddr := GlobalConfig.Clients[ID].Address
	myPort := GlobalConfig.Clients[ID].Port
	log.Println("ME")
	log.Println(myAddr)
	log.Println(myPort)

	// Start The Broadcaster
	go broadcaster()

	// Load Certs
	caCert, err := ioutil.ReadFile("./certs/ca.pem")
	if err != nil {
		log.Printf("failed to load cert: %s", err)
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	clientCertFile := fmt.Sprintf("./certs/client%d.pem", ID)
	clientKeyFile := fmt.Sprintf("./certs/client%d-key.pem", ID)
	clientCert, err := tls.LoadX509KeyPair(clientCertFile, clientKeyFile)

	// Set up TLS Configs
	tlsConfigServer := &tls.Config{
		Certificates: []tls.Certificate{clientCert},  // server certificate which is valIDated by the client
		ClientCAs:    caCertPool,                     // used to verify the client cert is signed by the CA and is therefore valID
		ClientAuth:   tls.RequireAndVerifyClientCert, // this requires a valID client certificate to be supplied during handshake
	}

	tlsConfigClient := &tls.Config{
		Certificates: []tls.Certificate{clientCert}, // this certificate is used to sign the handshake
		RootCAs:      caCertPool,                    // this is used to valIDate the server certificate
	}
	tlsConfigClient.BuildNameToCertificate()

	// Ring everyone else to start a connection with whoever is online
	for i, client := range GlobalConfig.Clients {
		if i == ID {
			continue
		}
		// Attempt to Dial
		log.Println("Ringing ", client.Port)
		conn, err := tls.Dial(
			"tcp", fmt.Sprintf("%s:%d", client.Address, client.Port), tlsConfigClient,
		)
		// If this worked use this as our communication chanel
		if err == nil {
			log.Println("Picked Up!")
			go handleConn(conn)
		}
	}

	// Start a listening server to pool connections when others come online
	go runListeningServer(myAddr, myPort, tlsConfigServer)
}

// Runs a server that accepts connections
func runListeningServer(myAddr string, myPort int, tlsConfigServer *tls.Config) {
	// Listen in the Port
	listener, err := tls.Listen("tcp", fmt.Sprintf("%s:%d", myAddr, myPort), tlsConfigServer)
	if err != nil {
		log.Fatal(err)
	}

	// Open connections with whoever rings
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Print(err)
			continue
		}
		log.Println("Call Received!")

		tlsConn, ok := conn.(*tls.Conn)
		if !ok {
			log.Printf("failed to cast conn to tls.Conn")
			continue
		}

		go handleConn(tlsConn)
	}
}

// What is run when any connection is established
func handleConn(conn *tls.Conn) {
	defer conn.Close()

	tag := fmt.Sprintf("[%s -> %s]", conn.LocalAddr(), conn.RemoteAddr())
	log.Printf("%s accept", tag)

	// Do the mTLS handshake
	err := conn.Handshake()
	if err != nil {
		log.Printf("failed to complete handshake: %s", err)
	}

	var clientName string
	if len(conn.ConnectionState().PeerCertificates) > 0 {
		clientName = conn.ConnectionState().PeerCertificates[0].Subject.CommonName
		log.Printf("%s client common name: %+v", tag, conn.ConnectionState().PeerCertificates[0].Subject.CommonName)
	} else {
		log.Printf("Error -- malformatted cert")
		return
	}

	// Start a channel and writer for outgoing messages
	ch := make(chan string)
	go clientWriter(conn, ch)

	// Register the channel with the broadcaster
	entering <- ch

	// Wrap incoming messages and send them in the IncomingMessages channel
	input := bufio.NewScanner(conn)
	for input.Scan() {
		IncomingMessages <- IncomingMessage{Client: clientName, Message: input.Text()}
	}
	log.Println("Done: ", strings.Split(conn.RemoteAddr().String(), ":")[1])
	leaving <- ch
}

// Writes outgoing messages to a channel
func clientWriter(conn *tls.Conn, ch <-chan string) {
	for msg := range ch {
		fmt.Fprintln(conn, msg) // NOTE: ignoring network errors
	}
}

// Broadcaster handles taking incoming messages and sending them on all connections. It also keeps track of
// all the connections. If you send a channel on entering, it will register that to send against. If you
// send a channel on leaving, it will recognize that channel is no longer valid, and remove that from
// those it broadcasts it to
func broadcaster() {
	clients := make(map[client]bool) // all connected clients
	for {
		select {
		case msg := <-Messages:
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
