package main

import (
	"fmt"
	"io"
	"log"
	"net"

	"github.com/golang/protobuf/proto"
	s "github.com/mimoo/sasayaki/serialization"
)

getMessage := &s.GetMessage{
			Type: proto.String("getlist"),
			Type:  proto.Int32(17),
			Reps:  []int64{1, 2, 3},
		}

func sasayakiServer(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("rpc server cannot accept client:", err)
			continue
		}

		handleClient(conn)
	}
}

func handleClient(conn net.Conn) {
	buffer := make([]byte, 3000)

	for {
		n, err := conn.Read(buffer)
		if err != nil {
			if err != io.EOF {
				log.Println("rpc server cannot read client request:", err)
			}
			break // always break on error
		}
		request := buffer[:n]
		fmt.Println(request) // should json parse that
		// should handle the request here
	}

	conn.Close()
}
