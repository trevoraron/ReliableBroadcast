package main

import (
	"fmt"
	"github.com/TrevorAron/ReliableBroadcast/config"
	"github.com/TrevorAron/ReliableBroadcast/connectionpool"
	"github.com/TrevorAron/ReliableBroadcast/helpers"
	"log"
	"math/rand"
	"strconv"
	"time"
)

type DataMessage struct {
	Message []byte
}

type IncomingMessage struct {
	Client string
	Data   DataMessage
}

// Global Channels
var (
	OutgoingMessages = make(chan DataMessage)     // Messages from the application
	IncomingMessages = make(chan IncomingMessage) // Messages to the application
)

// Sets Up Everything
func Setup() {
	// Seed the random number generator
	rand.Seed(time.Now().UTC().UnixNano())
	// Deal with messages coming from the connection pool
	go handleIncomingMessages()

	connectionpool.StartConnectionPool()

	// Deal with messages coming from the client
	go handleOutgoingMessages()
}

type DataStoreKey struct {
	Client int
	Nonce  int
}

// These are messages coming in from the connection pool
func handleIncomingMessages() {
	dataStore := map[DataStoreKey]*ProtocolState{}

	for incomingMessage := range connectionpool.IncomingMessages {
		// Find the client num
		client, err := clientNameToNumber(incomingMessage.Client)
		if err != nil {
			log.Fatalf("Error Parsing Client Name: %s", err)
		}
		// Now Decode the message
		var protoMessage ProtocolMessage
		err = helpers.BytesToStruct(incomingMessage.Data.Message, &protoMessage)
		if err != nil {
			log.Fatalf("Error Decoding Protocol Message: %s", err)
		}

		log.Printf("Received Message type %d from %s", protoMessage.MessageType, incomingMessage.Client)
		// Check if we have an entry in the DB for this
		dataStoreKey := DataStoreKey{Client: protoMessage.Broadcaster, Nonce: protoMessage.Nonce}
		state, ok := dataStore[dataStoreKey]
		// If not, make one
		if !ok {
			log.Printf("Creating Row for [%d,%d]", protoMessage.Broadcaster, protoMessage.Nonce)
			state, err = NewProtocolState(protoMessage.Nonce, protoMessage.Broadcaster)
			if err != nil {
				log.Fatalf("Error Creating Protocol State: %s", err)
			}
			// Add it into the DB
			dataStore[dataStoreKey] = state
		}

		// Run receive msg and see what we get
		payload, messages, err := state.ReceiveMsg(protoMessage, client)
		if err != nil {
			log.Fatalf("Error Recieving Msg: %s", err)
		}
		// Broadcast all messages
		for _, message := range messages {
			log.Printf("Broadcasting Message Type %d", message.MessageType)

			byteArr, err := helpers.StructToBytes(message)
			if err != nil {
				log.Fatalf("Error Encoding Proto Message: %s", err)
			}

			connectionpool.OutgoingMessages <- connectionpool.DataMessage{Message: byteArr}

		}

		// Commit if we can
		if payload != nil {
			// Convert Payload back to data message and send it up
			var dataMessage DataMessage
			err := helpers.BytesToStruct(payload, &dataMessage)
			if err != nil {
				log.Fatalf("Error Decoding Data Message: %s", err)
			}

			IncomingMessages <- IncomingMessage{
				Client: fmt.Sprintf("client%d", state.Broadcaster),
				Data:   dataMessage,
			}
		}
	}
}

// These are messages coming from the client
func handleOutgoingMessages() {
	for message := range OutgoingMessages {
		// Craft the broadcast message
		byteArr, err := helpers.StructToBytes(message)
		if err != nil {
			log.Fatalf("Error Encoding Outgoing Proto Message: %s", err)
		}
		protoMessage := ProtocolMessage{
			Nonce:       rand.Int(),
			Broadcaster: config.ID,
			MessageType: BROADCAST,
			Payload:     byteArr,
		}
		byteProtoMessage, err := helpers.StructToBytes(protoMessage)
		if err != nil {
			log.Fatalf("Error Encoding Outgoing Proto Message: %s", err)
		}

		dataMessage := connectionpool.DataMessage{Message: byteProtoMessage}

		// Send it to myself
		// TODO: get rid of this gross hack
		connectionpool.IncomingMessages <- connectionpool.IncomingMessage{
			Client: fmt.Sprintf("client%d", config.ID),
			Data:   dataMessage,
		}
		// Send it to everyone
		log.Printf("Broadcasting Message")
		connectionpool.OutgoingMessages <- dataMessage
	}
}

// Clients are named like 'client0'
func clientNameToNumber(client string) (int, error) {
	i, err := strconv.Atoi(client[6:])
	if err != nil {
		return 0, err
	}
	return i, nil
}
