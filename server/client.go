package main

import (
	"fmt"
	"io"
	"log"
	"net"

	"github.com/golang/protobuf/proto"
	s "github.com/mimoo/sasayaki/serialization"

	disco "github.com/mimoo/disco/libdisco"
)

type pendingMessage struct {
	fromKey string
	id      uint64
	convoId uint64
	content []byte
}

var pendingMessages map[string][]pendingMessage

func sasayakiServer(listener *disco.Listener) {
	for {
		conn, err := listener.AcceptDisco()
		if err != nil {
			log.Println("rpc server cannot accept client:", err)
			continue
		}
		log.Println("client accepted", conn.RemoteAddr().String())

		//		clientKey, err := disco.Conn(conn).RemotePublicKey()
		clientKey, err := conn.RemotePublicKey()
		if err != nil {
			log.Println("cannot read client public key:", err)
			conn.Close()
			continue
		}
		log.Println("client accepted", clientKey)

		go handleClient(conn)
	}
}

func handleClient(conn net.Conn) {
	//
	// TODO: is it necessary to create this huge buffer here?
	// don't proto have some functions to do that?
	//
	buffer := make([]byte, 10000)

	for {
		// read socket
		n, err := conn.Read(buffer)
		if err != nil {
			if err != io.EOF {
				log.Println("rpc server cannot read client request:", err)
			}
			break // always break on error
		}
		log.Println("received message from client")

		// parse protobuff request
		request := &s.Request{}
		err = proto.Unmarshal(buffer[:n], request)
		if err != nil {
			log.Println("unmarshaling error: ", err)
			break
		}

		switch request.GetRequestType() {
		case s.Request_GetPendingMessages:
			log.Println("client is requesting to get pending messages")
			/*
				am = []s.ResponseMessages_Message

				pm := pendingMessages[clientKey]

				resp := &s.ResponseMessages{
					Messages: am,
				}*/
		case s.Request_SendMessage:
			log.Println("client is requesting to send a message")
			handleSendMessage(conn, request)
		default:
			fmt.Println("request cannot be parsed yet")
			break
		}

	}

	log.Printf("%s closed the connection\n", conn.RemoteAddr().String())
	conn.Close()
}

func handleSendMessage(conn net.Conn, req *s.Request) {

	// write success or not

}
