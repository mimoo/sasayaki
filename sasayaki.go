//
// Sasayaki-Core
// =============
//
// This is the core state machine of the protocol, it:
// - is used by both the web UI and the CLI UI
// - communicates with the Hub via the HubManager to receive and send messages
// - removes or add encryption with the help of the Encryption Manager
//
//
package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"

	disco "github.com/mimoo/disco/libdisco"
)

type sasayakiState struct {
	keyPair   *disco.KeyPair // my long-term static keypair
	myAddress string         // public key in hex form

	initialized bool // useful for webUI

	debug bool // debug stuff
}

var ssyk sasayakiState

// getNextMessage retrieves and decrypt a new message from the hub
// everything message order is kept by the server
func (ss sasayakiState) getNextMessage() (*plaintextMsg, error) {
	// initialized?
	if !ssyk.initialized {
		return nil, errors.New("ssyk: Sasayaki has not been initialized")
	}
	// obtain next message from hub
	encryptedMsg, err := hub.getNextMessage()
	if err != nil {
		return nil, err
	}

	// check fields
	if len(encryptedMsg.GetConvoId()) != 32 {
		return nil, errors.New("ssyk: message received malformed")
	}

	// no address == no new message
	if encryptedMsg.GetFromAddress() == "" {
		return nil, errors.New("ssyk: no new messages")
	}

	// new convo? create it
	if !storage.ConvoExist(encryptedMsg.GetConvoId()) {
		if err := e2e.createConvoFromMessage(encryptedMsg); err != nil {
			return nil, err
		}
		return nil, nil // TODO: nil means new convo???
	}

	// TODO: do we care about checking the id field?

	// remove encryption
	decryptedMessage, err := e2e.decryptMessage(encryptedMsg)
	if err != nil {
		return nil, err
	}

	// store message
	storage.storeMessage(decryptedMessage)

	// TODO: tell the server it can safely delete tuple with {id, convoId, bobAddress}
	// 			but isn't that going to be way too large messages? That could be sent on the notification channel
	// 			the notif channel could be a two way channel

	// returns
	return decryptedMessage, nil
}

// sendMessage can be used to send a message, or create a new thread
// in the case of a new thread, convoId must be "0" and the content must be the thread's title
func (ss sasayakiState) sendMessage(msg *plaintextMsg) (string, error) {
	// initialized?
	if !ssyk.initialized {
		return "", errors.New("ssyk: Sasayaki has not been initialized")
	}
	// is it a new thread?
	if msg.ConvoId == "" {
		// generate convoId
		var randomBytes [16]byte
		if _, err := rand.Read(randomBytes[:]); err != nil {
			panic(err)
		}
		msg.ConvoId = hex.EncodeToString(randomBytes[:])

		// create new convo + store
		encryptedMessage, err := e2e.createNewConvo(msg)
		if err != nil {
			return "", err
		}

		// send to hub
		if err := hub.sendMessage(encryptedMessage); err != nil {
			return "", err
		}
	} else { // nope, it's just a message
		// add encryption
		encryptedMessage, err := e2e.encryptMessage(msg)
		if err != nil {
			return "", err
		}

		// send to hub
		if err := hub.sendMessage(encryptedMessage); err != nil {
			return "", err
		}

		// store in database
		storage.storeMessage(msg)
	}

	//
	return msg.ConvoId, nil
}

func (ss sasayakiState) addContact(bobAddress string) error {
	panic("not implemented")
	if len(bobAddress) != 64 {
		return errors.New("ssyk: contact's address is malformed")
	}

	bobPubKey, err := hex.DecodeString(bobAddress)
	if err != nil {
		return errors.New("ssyk: contact's address is not hexadecimal")
	}

	return e2e.addContact(bobPubKey)
}

func (ss sasayakiState) acceptContact(bobAddress string, firstHandshakeMessage []byte) error {
	panic("not implemented")
	if len(bobAddress) != 64 {
		return errors.New("ssyk: contact's address is malformed")
	}

	bobPubKey, err := hex.DecodeString(bobAddress)
	if err != nil {
		return errors.New("ssyk: contact's address is not hexadecimal")
	}

	return e2e.finishHandshake(bobPubKey, firstHandshakeMessage)
}
