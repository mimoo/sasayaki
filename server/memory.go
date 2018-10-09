// This is an in-memory database
// for testing only
package main

type Message struct {
	fromAddress string
	id          uint64
	convoId     uint64
	content     []byte
}

var (
	pendingMessages map[string][]Message // in-memory pending messages (for testing)
)

func init() {
	pendingMessages = make(map[string][]Message)
}
