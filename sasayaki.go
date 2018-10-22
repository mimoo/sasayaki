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
// Ideally:
// - all checks on inputs should be done here, not before
// - core calls the storage service, the hub service and the e2e encryption service itself (not always true atm)
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
var errNotInitialized = errors.New("ssyk: Sasayaki has not been initialized")

// getNextMessage retrieves and decrypt a new message from the hub
// everything message order is kept by the server
func (ss sasayakiState) getNextMessage() (*plaintextMsg, error) {
	// initialized?
	if !ssyk.initialized {
		return nil, errNotInitialized
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
	// 			honestly, if we fail soemthing on our side it's probable that we won't be able to recover anyway
	// 			(so better to just delete things on the server side)

	// returns
	return decryptedMessage, nil
}

// sendMessage can be used to send a message, or create a new thread
// in the case of a new thread, convoId must be "0" and the content must be the thread's title
func (ss sasayakiState) sendMessage(msg *plaintextMsg) (string, error) {
	// initialized?
	if !ssyk.initialized {
		return "", errNotInitialized
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

// TODO: what happens when we do that?
// should it send a message coming from us? Or a meta msg from the hub?
// maybe if we receive a msg from someone we don't know, we can assume it is a request
func (ss sasayakiState) addContact(bobAddress, bobName string) error {
	panic("not implemented")
	// initialized?
	if !ssyk.initialized {
		return nil, errNotInitialized
	}
	if len(bobAddress) != 64 {
		return errors.New("ssyk: contact's address is malformed")
	}

	// TODO: check that we don't already have the contact
	// if we do, what to do? refresh for future secrecy?

	firstHandshakeMessage := e2e.addContact(bobAddress, bobName)

	// TODO: forward request to hub
}

// TODO: we should be able to acceptContact even if we did it in the past (
// for example alice could do addContact / deleteContact / addContact
// TODO: should it return the contact id or something usable by the app?
func (ss sasayakiState) acceptContact(aliceAddress, aliceName string, firstHandshakeMessage []byte) error {
	panic("not implemented")
	// initialized?
	if !ssyk.initialized {
		return nil, errNotInitialized
	}
	// TODO: move all these checks in ssyk?
	if len(aliceAddress) != 64 {
		return errors.New("ssyk: contact's address is malformed")
	}

	return e2e.acceptContact(aliceAddress, aliceName, firstHandshakeMessage)
}

// TODO: we need to receive the information that bob has accepted our contact request
// how? via receipt of a "meta" message?
// how are friend requests sent anyway?
func (ss sasayakiState) ackAcceptContact(bobAddress string, secondHandshakeMessage []byte) error {
	panic("not implemented")
	// initialized?
	if !ssyk.initialized {
		return errNotInitialized
	}

	return e2e.finishHandshake(bobAddress, secondHandshakeMessage)
}

// deleteContact is used to delete a contact from storage
func (ss sasayakiState) deleteContact(bobAddress string) error {
	panic("not implemented")
	// initialized?
	if !ssyk.initialized {
		return nil, errNotInitialized
	}

	return storage.deleteContact(bobAddress)
}
