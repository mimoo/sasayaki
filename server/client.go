package main

import (
	"fmt"
	"io"
	"log"
	"net"

	"github.com/golang/protobuf/proto"
	s "github.com/mimoo/sasayaki/serialization"
)

type pendingMessage struct {
	fromKey string
	id      uint64
	convoId uint64
	content []byte
}

var pendingMessages map[string]pendingMessage

func sasayakiServer(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("rpc server cannot accept client:", err)
			continue
		}
		log.Println("client accepted", conn.RemoteAddr().String())

		go handleClient(conn)
	}
}

func handleClient(conn net.Conn) {
	buffer := make([]byte, 3000)

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

		// ...
		fmt.Println(request) // should parse that
		// should handle the request here
		resp := &s.ResponseMessages{}
		fmt.Println(resp)
	}

	log.Printf("%s closed the connection\n", conn.RemoteAddr().String())
	conn.Close()
}
