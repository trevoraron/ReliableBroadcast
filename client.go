package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
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

	myAddr := config.Clients[id].Address
	myPort := config.Clients[id].Port
	fmt.Println("ME")
	fmt.Println(myAddr)
	fmt.Println(myPort)

	other := (id + 1) % 2
	otherAddr := config.Clients[other].Address
	otherPort := config.Clients[other].Port
	fmt.Println("OTHER")
	fmt.Println(myAddr)
	fmt.Println(myPort)

	done := make(chan struct{})

	// Attempt to Dial
	fmt.Println("Ringing")
	conn, err := net.Dial(
		"tcp", fmt.Sprintf("%s:%d", otherAddr, otherPort),
	)
	// If this worked use this as our communication chanel
	if err == nil {
		fmt.Println("Picked Up!")
		go func() {
			io.Copy(os.Stdout, conn) // NOTE: ignoring errors
			log.Println("done")
			done <- struct{}{} // signal the main goroutine
		}()
		mustCopy(conn, os.Stdin)
		conn.Close()
		<-done
		// Otherwise Create a listening server -- we hopped on first
	} else {
		fmt.Println("Failed. Listening")
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
}

type Config struct {
	Clients []struct {
		Address string
		Port    int
	}
}

func handleConn(conn net.Conn) {
	done := make(chan struct{})
	go func() {
		io.Copy(os.Stdout, conn) // NOTE: ignoring errors
		log.Println("done")
		done <- struct{}{} // signal the main goroutine
	}()
	mustCopy(conn, os.Stdin)
	<-done
	// NOTE: ignoring potential errors from input.Err()
	conn.Close()
}

func mustCopy(dst io.Writer, src io.Reader) {
	if _, err := io.Copy(dst, src); err != nil {
		log.Fatal(err)
	}
}
