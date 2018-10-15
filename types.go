//
// Types
// =====
//
// This file contains the different types of Sasayaki, so that different parts of the program
// (called managers) can communicate via these types
//
package main

// plaintextMessage is plaintext-message type
// It is important to use a special type for plaintext messages as protobuffer messages could set
// the decrypted content directly or other logic bugs might arise
type plaintextMsg struct {
	ConvoId     string `json:"convo_id"`
	FromAddress string `json:"from_address"`
	ToAddress   string `json:"to_address"`

	Content string `json:"content"`
}
