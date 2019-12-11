package main

import (
	"bytes"
	"fmt"
	"github.com/TrevorAron/ReliableBroadcast/config"
	"github.com/pkg/errors"
	"math"
)

// Types sent over the Wire
type MessageType int32

const (
	BROADCAST MessageType = 0
	ECHO      MessageType = 1
	READY     MessageType = 2
)

type ProtocolMessage struct {
	Nonce       int
	Broadcaster int
	MessageType MessageType
	Payload     []byte
}

type ProtocolState struct {
	Payload          []byte
	Broadcaster      int
	Nonce            int
	SentEcho         bool
	SentReady        bool
	CommittedPayload bool
	EchoMessages     [][]byte
	ReadyMessages    [][]byte
}

var ErrUnknownMessageType = errors.New("unknown protocol message type")
var ErrInvalidClientMsg = errors.New("client sent a message it shouldn't have")

// Crafts a new protocol state based on an incoming message from a client
func NewProtocolState(nonce int, broadcaster int) (*ProtocolState, error) {
	state := ProtocolState{
		EchoMessages:     make([][]byte, getN()),
		ReadyMessages:    make([][]byte, getN()),
		SentEcho:         false,
		SentReady:        false,
		CommittedPayload: false,
		Payload:          nil,
	}
	// Clear the messages array
	for i := 0; i < getN(); i++ {
		state.EchoMessages[i] = nil
		state.ReadyMessages[i] = nil
	}

	state.Nonce = nonce
	state.Broadcaster = broadcaster

	return &state, nil
}

// See if we have enough matching echo messages (n + t + 1) / 2 to send ready
func (s *ProtocolState) enoughEchoMsgsForReady() []byte {
	t := config.GlobalConfig.T
	n := getN()
	threshold := int(math.Ceil(float64(n+t+1) / float64(2)))
	return checkForThreshold(s.EchoMessages, threshold)
}

// See if we have enough matching ready messages (t + 1) to send ready
func (s *ProtocolState) enoughReadyMsgsForReady() []byte {
	t := config.GlobalConfig.T
	return checkForThreshold(s.ReadyMessages, t+1)
}

// See if we have enough matching ready messages (t + 1) to send ready
func (s *ProtocolState) enoughReadyMsgsForCommit() []byte {
	t := config.GlobalConfig.T
	return checkForThreshold(s.ReadyMessages, 2*t+1)
}

// Will Return Payload if we can commit
// Will Return a list of Msgs to broadcast, if any
func (s *ProtocolState) ReceiveMsg(msg ProtocolMessage, client int) ([]byte, []ProtocolMessage, error) {
	var msgsToSend []ProtocolMessage
	// First we update state
	switch msg.MessageType {
	case BROADCAST:
		if client != msg.Broadcaster {
			return nil, nil, errors.Wrap(
				ErrInvalidClientMsg, fmt.Sprintf("Client: %d, Broadcaster: %d", client, msg.Broadcaster),
			)
		}
		s.Payload = msg.Payload
	case ECHO:
		s.EchoMessages[client] = msg.Payload
	case READY:
		s.ReadyMessages[client] = msg.Payload
	default:
		return nil, nil, errors.Wrap(ErrUnknownMessageType, fmt.Sprintf("Msg Type: %d", msg.MessageType))
	}

	// See if we can send an Echo
	if !s.SentEcho && s.Payload != nil {
		msgsToSend = append(
			msgsToSend,
			ProtocolMessage{Nonce: s.Nonce, Payload: s.Payload, Broadcaster: s.Broadcaster, MessageType: ECHO},
		)
		s.EchoMessages[config.ID] = s.Payload
		s.SentEcho = true
	}

	// See if we can send a ready
	if !s.SentReady {
		// We need either a threshold of Echos or Readys
		payload := s.enoughEchoMsgsForReady()
		if payload == nil {
			payload = s.enoughReadyMsgsForReady()
		}

		// We have a threshold of either echos or readys so lets send a ready
		if payload != nil {
			msgsToSend = append(
				msgsToSend,
				ProtocolMessage{Nonce: s.Nonce, Payload: payload, Broadcaster: s.Broadcaster, MessageType: READY},
			)
			s.ReadyMessages[config.ID] = payload
			s.SentReady = true
		}
	}

	// Finally, see if we can deliver the payload
	commitPayload := s.enoughReadyMsgsForCommit()
	// We should only deliver to the application once
	if s.CommittedPayload {
		commitPayload = nil
	}
	if commitPayload != nil {
		s.CommittedPayload = true
	}

	return commitPayload, msgsToSend, nil
}

// Gets N based on the Config
func getN() int {
	return len(config.GlobalConfig.Clients)
}

// Returns if there is a payload of a threshold of matching values
func checkForThreshold(arr [][]byte, threshold int) []byte {
	// Loop through twice. For each value, if it is not nill, see if there is a threshold of values
	for i := 0; i < len(arr); i++ {
		if arr[i] == nil {
			continue
		}
		matching := 1
		for j := 0; j < len(arr); j++ {
			if i == j {
				continue
			}

			if bytes.Compare(arr[i], arr[j]) == 0 {
				matching++
			}
		}
		if matching >= threshold {
			return arr[i]
		}
	}

	return nil
}
