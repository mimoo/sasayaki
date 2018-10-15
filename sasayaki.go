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
	"errors"
	"log"

	disco "github.com/mimoo/disco/libdisco"
)

type sasayakiState struct {
	keyPair   *disco.KeyPair // my long-term static keypair
	myAddress string         // public key in hex form

	initialized bool // useful for webUI
}

var ssyk sasayakiState

func (ss sasayakiState) getNextMessage() (*plaintextMsg, error) {
	// obtain next message from hub
	encryptedMsg, err := hub.getNextMessage()
	if err != nil {
		return nil, err
	}

	// no address == no new message
	if encryptedMsg.GetFromAddress() == "" {
		return nil, errors.New("no new messages")
	}

	// new convo? create it
	if encryptedMsg.GetId() == 1 {
		if err := e2e.createConvoFromMessage(encryptedMsg); err != nil {
			return nil, err
		}
		return nil, nil // TODO: nil means new convo???
	}

	// TODO: do we care about checking the id field? or convoId field?

	// remove encryption
	decryptedMessage, err := e2e.decryptMessage(encryptedMsg)
	if err != nil {
		return nil, err
	}

	// store message
	id := storage.storeMessage(decryptedMessage)

	// TODO: do we care about checking the id field here?
	if id != decryptedMessage.Id {
		log.Println("ssyk: wrong id received. Do we care?")
	}

	// TODO: tell the server it can safely delete tuple with {id, convoId, bobAddress}
	// 			but isn't that going to be way too large messages? That could be sent on the notification channel
	// 			the notif channel could be a two way channel

	// returns
	return decryptedMessage, nil
}

func (ss sasayakiState) sendMessage(msg *plaintextMsg) error {

	// add to database + update id
	msg.Id = storage.storeMessage(msg)

	// add encryption
	encryptedMessage, err := e2e.encryptMessage(msg)
	if err != nil {
		panic(err)
	}

	// go through proxy
	return hub.sendMessage(encryptedMessage)
}
