//
// Notification Service
// ====================
// 
// This is a two-way communication channel where:
// - users can notify the server that they have read a message
// - the server can notify the client that they have received a new message
//
// This is in contrast with the primary delivery service which is a simple JSON REST API
//
package main

import (
	"log"
	"net"
)

type notifClient struct {
	conn      net.Conn
	publicKey string
}

func notificationClient(conn net.Conn) {
	// triggers the handshake
	conn.Write([]byte{})
	// info
	log.Println("client accepted", conn.RemoteAddr().String())
	// get client's pubkey
	clientKey, err := conn.RemotePublicKey()
	if err != nil {
		log.Println("cannot read client public key:", err)
		conn.Close()
		continue
	}
	log.Println("client accepted", clientKey)

	cc := notifClient{
		conn:      conn,
		publicKey: clientKey,
	}

	go cc.handleNotificationsFromClient()
	cc.handleDistributionToClient()

	//
	conn.Close()
}

func (nc notifClient) handleNotificationsFromClient() {
	for {

	}
}

func (nc notifClient) handleDistributionToClient() {
	for {
		select
	}
}
