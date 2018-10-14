//
// Hub Manager
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
	"encoding/hex"
	"errors"
	"net"
	"sync"

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
	conn       net.Conn   // the connection to the hub
	queryMutex sync.Mutex // one hub query at a time
}

var hub hubState

func initHubManager() error {
	// if we already have a conn, return
	if hub.conn != nil {
		return nil
	}
	// decode the hub public key
	hubPublicKey, err := hex.DecodeString(ssyk.config.HubPublicKey)
	if err != nil {
		return err
	}
	// config for IK handshake
	clientConfig := disco.Config{
		KeyPair:              ssyk.keyPair,
		HandshakePattern:     disco.Noise_IK,
		RemoteKey:            hubPublicKey,
		StaticPublicKeyProof: []byte{},
	}
	// dial the Hub and set `conn`
	hub.conn, err = disco.Dial("tcp", ssyk.config.HubAddress, &clientConfig)
	if err != nil {
		return err
	}

	return nil
}

// TODO: of course encrypt the message before sending it :)
// TODO: needs a cryptoManager? or endToEndManager? or encryptionManager
func (hub *hubState) sendMessage(id, convoId uint64, toAddress string, content []byte) (bool, string) {
	// check for arbitrary 1000 bytes of room for headers and protobuff structure
	if len(content) > 65535-1000 {
		return false, errors.New("ssyk: message to send is too large")
	}
	// one query at a time
	hub.queryMutex.Lock()
	defer hub.queryMutex.Unlock()
	// do we have a connection working?
	if err := initHubManager(); err != nil {
		return false, err.Error()
	}
	// create query
	req := &s.Request{
		RequestType: s.Request_SendMessage,
		Message: &s.Request_Message{
			ToAddress: toAddress,
			Id:        id,
			ConvoId:   convoId,
			Content:   content,
		},
	}
	// serialize
	data, err := proto.Marshal(req)
	if err != nil {
		return false, err.Error()
	}
	// encode [length(2), data(...)]
	data = append([]byte{byte(len(data) >> 8), byte(len(data))}, data...)
	// send
	if _, err := hub.conn.Write(data); err != nil {
		hub.conn = nil
		return false, err.Error()
	}
	// receive header
	var header [2]byte
	n, err := hub.conn.Read(header[:])
	if err != nil || n != 2 {
		hub.conn = nil
		return false, err.Error()
	}
	length := (header[0] << 8) | header[1]
	// receive
	rcvBuffer := make([]byte, length)
	n, err = hub.conn.Read(rcvBuffer)
	if err != nil {
		hub.conn = nil
		return false, err.Error()
	}
	// unserialize
	res := &s.ResponseSuccess{}
	if err = proto.Unmarshal(rcvBuffer[:n], res); err != nil {
		return false, err.Error()
	}
	// return
	return res.GetSuccess(), res.GetError()
}

func (hub *hubState) getNextMessage() (*s.ResponseMessage, error) {
	// one query at a time
	hub.queryMutex.Lock()
	defer hub.queryMutex.Unlock()
	// do we have a connection?
	initHubManager()
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
	// return message
	return res, nil
}
