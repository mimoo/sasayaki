//
// Hub Proxy
// ===========
//
// This file forwards requests to the Hub, to do that it:
//
// * serializes requests with protobuf
// * send them to the hub (ip is from config file)
// * unserialize the protobuf response
//
package main

import (
	"errors"
	"net"

	"github.com/golang/protobuf/proto"

	s "github.com/mimoo/sasayaki/serialization"

	disco "github.com/mimoo/disco/libdisco"
)

const (
	maxConnectionAttempts = 5
)

var (
	rcvBuffer [10000]byte
)

type hubState struct {
	conn net.Conn // the connection to the hub

	hubAddress   string
	hubPublicKey []byte
}

func initHubState(hubAddress string, hubPublicKey []byte) *hubState {
	hub := &hubState{
		hubAddress:   hubAddress,
		hubPublicKey: hubPublicKey,
	}
	return hub
}

func (hub *hubState) isHubReady() error {
	// if we already have a conn, return
	if hub.conn != nil {
		return nil
	}
	// decode the hub public key
	if hub.hubAddress == "" || hub.hubPublicKey == nil {
		return errors.New("Hub not properly configured")
	}
	// config for IK handshake
	clientConfig := disco.Config{
		KeyPair:              ssyk.keyPair,
		HandshakePattern:     disco.Noise_IK,
		RemoteKey:            hub.hubPublicKey,
		StaticPublicKeyProof: []byte{},
	}
	// dial the Hub and set `conn`
	var err error
	hub.conn, err = disco.Dial("tcp", hub.hubAddress, &clientConfig)
	if err != nil {
		return err
	}

	return nil
}

// TODO: of course encrypt the message before sending it :)
// TODO: needs a cryptoManager? or endToEndManager? or encryptionManager
func (hub *hubState) sendMessage(encryptedMessage *s.Request_Message) error {
	// do we have a connection working?
	if err := hub.isHubReady(); err != nil {
		return err
	}
	// proto structure
	req := &s.Request{
		RequestType: s.Request_SendMessage,
		Message:     encryptedMessage,
	}

	// serialize
	data, err := proto.Marshal(req)
	if err != nil {
		return err
	}
	// encode [length(2), data(...)]
	data = append([]byte{byte(len(data) >> 8), byte(len(data))}, data...)
	// send
	if _, err := hub.conn.Write(data); err != nil {
		hub.conn = nil
		return err
	}
	// receive header
	var header [2]byte
	n, err := hub.conn.Read(header[:])
	if err != nil || n != 2 {
		hub.conn = nil
		return err
	}
	length := (header[0] << 8) | header[1]
	// receive
	rcvBuffer := make([]byte, length)
	n, err = hub.conn.Read(rcvBuffer)
	if err != nil {
		hub.conn = nil
		return err
	}
	// unserialize
	res := &s.ResponseSuccess{}
	if err = proto.Unmarshal(rcvBuffer[:n], res); err != nil {
		return err
	}

	// return on failure
	if !res.GetSuccess() {
		return errors.New(res.GetError())
	}

	return nil
}

// getNextMessage receives a protobuffer structure and returns a message type
func (hub *hubState) getNextMessage() (*s.ResponseMessage, error) {
	// do we have a connection?
	if err := hub.isHubReady(); err != nil {
		return nil, err
	}
	// create query
	req := &s.Request{RequestType: s.Request_GetNextMessage}
	// serialize
	data, err := proto.Marshal(req)
	if err != nil {
		return nil, err
	}
	// encode [length(2), data(...)]
	data = append([]byte{byte(len(data) >> 8), byte(len(data))}, data...)
	// send
	if _, err = hub.conn.Write(data); err != nil {
		hub.conn = nil
		return nil, err
	}
	// receive header
	var header [2]byte
	n, err := hub.conn.Read(header[:])
	if err != nil || n != 2 {
		hub.conn = nil
		return nil, err
	}
	length := (header[0] << 8) | header[1]
	// receive
	rcvBuffer := make([]byte, length)
	n, err = hub.conn.Read(rcvBuffer)
	if err != nil {
		hub.conn = nil
		return nil, err
	}
	// unserialize
	res := &s.ResponseMessage{}
	if err = proto.Unmarshal(rcvBuffer[:n], res); err != nil {
		return nil, err
	}

	// TODO: make sure that fields are set here?

	// return message
	return res, nil
}
