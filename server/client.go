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

session:
	for {
		// receive header
		var header [2]byte
		n, err := conn.Read(header[:])
		if err != nil || n != 2 {
			log.Println("can't read header: ", err)
			break session
		}
		length := (header[0] << 8) | header[1]
		// receive
		buffer := make([]byte, length)
		// read socket
		n, err = conn.Read(buffer)
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
		var responseData []byte

		switch request.GetRequestType() {
		case s.Request_GetNextMessage:
			log.Println("client is requesting to get next message")
			responseData, err = cc.handleGetNextMessage(request)
			if err != nil {
				log.Println("client session closing:", err)
				break session
			}
		case s.Request_SendMessage:
			log.Println("client is requesting to send a message")
			responseData, err = cc.handleSendMessage(request)
			if err != nil {
				log.Println("client session closing:", err)
				break session
			}
		default:
			log.Println("request cannot be parsed yet")
			break session
		}

		// handle response, encode [length(2), data(...)]
		responseData = append([]byte{byte(len(responseData) >> 8), byte(len(responseData))}, responseData...)

		// send it
		_, err = conn.Write(responseData)
		if err != nil {
			log.Println("client session closing:", err)
			break session
		}

	}

	log.Printf("%s closed the connection\n", conn.RemoteAddr().String())
	conn.Close()
}

// success sends a failure or success proto message. Returns an error if it can't write to the conn
func success(success bool, message string) ([]byte, error) {
	res := &s.ResponseSuccess{
		Success: success,
		Error:   message,
	}
	return proto.Marshal(res)
}

// handleSendMessage attempts to send the message. Returns an error if it doesn't work
// because of conn. Otherwise send a failure proto message
func (cc client) handleSendMessage(req *s.Request) ([]byte, error) {
	// parse request
	convoId := req.Message.GetConvoId()
	toAddress := req.Message.GetToAddress()
	content := req.Message.GetContent()
	// checking fields
	// TODO: test if id or convo id = 0 ? (not set)
	if len(toAddress) != 64 || content == nil || len(content) > messageMaxChars || len(convoId) != 32 {
		return success(false, "fields are not correctly formated")
	}
	if !regexHex.MatchString(toAddress) {
		return success(false, "the recipient address is not [a-z0-9]")
	}
	toAddress = strings.ToLower(toAddress)
	// handle the message (TODO: do it w/ a database)

	mm.queryMutex.Lock()
	mm.pendingMessages[toAddress] = append(mm.pendingMessages[toAddress], Message{
		fromAddress: cc.publicKey,
		convoId:     convoId,
		content:     content,
	})
	mm.queryMutex.Unlock()

	// write success or not
	return success(true, "")
}

func (cc client) handleGetNextMessage(req *s.Request) ([]byte, error) {
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
		res.ConvoId = message.convoId
		res.Content = message.content
	}
	// unlock memory
	mm.queryMutex.Unlock()
	// serialize
	data, err := proto.Marshal(res)
	if err != nil {
		panic(err) // TODO: can this panic be triggered maliciously?
	}
	//
	return data, err
}
