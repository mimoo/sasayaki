// This is an in-memory database
// for testing only
package main

import "sync"

type memory struct {
	pendingMessages map[string][]Message // in-memory pending messages (for testing)
	queryMutex      sync.Mutex           // one query at a time
}

type Message struct {
	fromAddress string
	convoId     string
	content     []byte
}

var (
	mm memory
)

func init() {
	mm.pendingMessages = make(map[string][]Message)
}
