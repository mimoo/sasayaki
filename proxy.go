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
	"fmt"
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

var hs hubState

func initHubManager() error {
	// if we already have a conn, return
	if hs.conn != nil {
		return nil
	}
	// decode the hub public key
	hubPublicKey, err := hex.DecodeString(ss.config.HubPublicKey)
	if err != nil {
		return err
	}
	// config for IK handshake
	clientConfig := disco.Config{
		KeyPair:              ss.keyPair,
		HandshakePattern:     disco.Noise_IK,
		RemoteKey:            hubPublicKey,
		StaticPublicKeyProof: []byte{},
	}
	// dial the Hub and set `conn`
	hs.conn, err = disco.Dial("tcp", ss.config.HubAddress, &clientConfig)
	if err != nil {
		return err
	}

	return nil
}

// TODO: of course encrypt the message before sending it :)
// TODO: needs a cryptoManager? or endToEndManager? or encryptionManager
func (hs *hubState) sendMessage(id, convoId uint64, toAddress string, content []byte) (bool, string) {
	// one query at a time
	hs.queryMutex.Lock()
	defer hs.queryMutex.Unlock()
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
	fmt.Println("sending", req)
	// send
	if _, err := hs.conn.Write(data); err != nil {
		hs.conn = nil
		return false, err.Error()
	}
	// receive
	n, err := hs.conn.Read(rcvBuffer[:])
	if err != nil {
		hs.conn = nil
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

func (hs *hubState) getNextMessage() (*s.ResponseMessage, error) {
	// one query at a time
	hs.queryMutex.Lock()
	defer hs.queryMutex.Unlock()
	// do we have a connection?
	initHubManager()
	// create query
	req := &s.Request{RequestType: s.Request_GetNextMessage}
	// serialize
	data, err := proto.Marshal(req)
	if err != nil {
		return nil, err
	}
	// send
	if _, err = hs.conn.Write(data); err != nil {
		hs.conn = nil
		return nil, err
	}
	// receive
	n, err := hs.conn.Read(rcvBuffer[:])
	if err != nil {
		hs.conn = nil
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
