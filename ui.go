package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"net"

	"github.com/golang/protobuf/proto"

	s "github.com/mimoo/sasayaki/serialization"

	disco "github.com/mimoo/disco/libdisco"
)

func connectToHub() (net.Conn, error) {
	// if we already have a connection to the hub
	if state.conn != nil {
		return state.conn, nil
	}

	// TODO: remove these defaults
	if len(state.config.HubPublicKey) == 0 {
		state.config.HubPublicKey = "1274e5b61840d54271e4144b80edc5af946a970ef1d84329368d1ec381ba2e21"
	}
	if state.config.HubAddress == "" {
		state.config.HubAddress = "127.0.0.1:7474"
	}

	hubPublicKey, err := hex.DecodeString(state.config.HubPublicKey)
	if err != nil {
		return nil, fmt.Errorf("sasayaki: can't parse the given hub address (%s)\n", state.config.HubPublicKey)
	}

	// config
	clientConfig := disco.Config{
		KeyPair:              state.keyPair,
		HandshakePattern:     disco.Noise_IK,
		RemoteKey:            hubPublicKey,
		StaticPublicKeyProof: []byte{},
	}

	// dial
	return disco.Dial("tcp", state.config.HubPublicKey, &clientConfig)
}

// makeQuery attempts to make a query by first connecting to the hub if it hasn't already yet.
// if it does not work, it nullifies the connection and returns an error
func makeQuery(query []byte) ([]byte, error) {

	state.queryMutex.Lock()
	defer state.queryMutex.Unlock()

	conn, err := connectToHub()
	if err != nil {
		return nil, err
	}

	req := &s.Request{RequestType: s.Request_GetPendingMessages}

	data, err := proto.Marshal(req)
	fmt.Println("going to send", data)
	if err != nil {
		log.Panic("marshaling error: ", err)
	}
	_, err = conn.Write(data)

	if err != nil {
		conn = nil
		return nil, err
	}

	var buffer [3000]byte
	n, err := conn.Read(buffer[:])
	if err != nil {
		conn = nil
		return nil, err
	}
	fmt.Println("response received:", buffer[:n])

	// return response
	return buffer[:n], nil

	// we never attempt to close the connection ourselves
	// conn.Close()

}
