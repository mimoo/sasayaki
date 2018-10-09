package main

import (
	"io"
	"log"
	"net"
	"regexp"
	"strings"

	"github.com/golang/protobuf/proto"
	s "github.com/mimoo/sasayaki/serialization"

	disco "github.com/mimoo/disco/libdisco"
)

const (
	messageMaxChars = 10000
)

type client struct {
	publicKey string
}

var (
	regexHex *regexp.Regexp // regex to test if a string is a hex string
)

func init() {
	regexHex = regexp.MustCompile(`^[A-Fa-f0-9]+$`)
}

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

		cc := client{
			publicKey: clientKey,
		}

		go cc.handleClient(conn)
	}
}

func (cc client) handleClient(conn net.Conn) {
	//
	// TODO: is it necessary to create this huge buffer here?
	// don't proto have some functions to do that?
	// worst case perhaps I should use a pool of buffer (see crypto/tls' block)
	//
	buffer := make([]byte, 10000)

session:
	for {
		// read socket
		n, err := conn.Read(buffer)
		if err != nil {
			if err != io.EOF {
				log.Println("rpc server cannot read client request:", err)
			}
			break session // always break on error
		}
		log.Println("received message from client")
		// parse protobuff request
		request := &s.Request{}
		err = proto.Unmarshal(buffer[:n], request)
		if err != nil {
			log.Println("unmarshaling error: ", err)
			break session
		}
		// what kind of request?
		switch request.GetRequestType() {
		case s.Request_GetNextMessage:
			log.Println("client is requesting to get next message")
			if err := cc.handleGetNextMessage(conn, request); err != nil {
				log.Println("client session closing:", err)
				break session
			}
		case s.Request_SendMessage:
			log.Println("client is requesting to send a message")
			if err := cc.handleSendMessage(conn, request); err != nil {
				log.Println("client session closing:", err)
				break session
			}
		default:
			log.Println("request cannot be parsed yet")
			break session
		}

	}

	log.Printf("%s closed the connection\n", conn.RemoteAddr().String())
	conn.Close()
}

// success sends a failure or success proto message. Returns an error if it can't write to the conn
func success(conn net.Conn, success bool, message string) error {
	res := &s.ResponseSuccess{
		Success: success,
		Error:   message,
	}
	data, err := proto.Marshal(res)
	if err != nil {
		panic(err)
	}
	_, err = conn.Write(data)
	return err
}

// handleSendMessage attempts to send the message. Returns an error if it doesn't work
// because of conn. Otherwise send a failure proto message
func (cc client) handleSendMessage(conn net.Conn, req *s.Request) error {
	// parse request
	id := req.Message.GetId()
	convoId := req.Message.GetConvoId()
	toAddress := req.Message.GetToAddress()
	content := req.Message.GetContent()
	// checking fields
	// TODO: test if id or convo id = 0 ? (not set)
	if len(toAddress) != 64 || content == nil || len(content) > messageMaxChars {
		return success(conn, false, "fields are not correctly formated")
	}
	if !regexHex.MatchString(toAddress) {
		return success(conn, false, "the recipient address is not [a-z0-9]")
	}
	toAddress = strings.ToLower(toAddress)
	// handle the message (TODO: do it w/ a database)

	mm.queryMutex.Lock()
	mm.pendingMessages[toAddress] = append(mm.pendingMessages[toAddress], Message{
		fromAddress: cc.publicKey,
		id:          id,
		convoId:     convoId,
		content:     content,
	})
	mm.queryMutex.Unlock()

	// write success or not
	return success(conn, true, "")
}

func (cc client) handleGetNextMessage(conn net.Conn, req *s.Request) error {
	// empty response for now
	res := &s.ResponseMessage{}
	// lock memory
	mm.queryMutex.Lock()
	// fetch new message*s* (TODO: do it with a real db)
	messages, ok := mm.pendingMessages[cc.publicKey]
	// is there a new message?
	if ok && len(messages) > 0 {
		// fetch new message
		message := messages[0]
		mm.pendingMessages[cc.publicKey] = messages[1:]
		res.FromAddress = message.fromAddress
		res.Id = message.id
		res.ConvoId = message.convoId
		res.Content = message.content
	} else {
		res.FromAddress = "empty"
	}
	// unlock memory
	mm.queryMutex.Unlock()
	// serialize
	data, err := proto.Marshal(res)
	if err != nil {
		panic(err) // TODO: can this panic be triggered maliciously?
	}
	// send
	_, err = conn.Write(data)
	return err
}
