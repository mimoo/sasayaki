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
	"sync"

	disco "github.com/mimoo/disco/libdisco"
)

type sasayakiState struct {
	myAddress string // public key in hex form

	queryMutex sync.Mutex // one query at a time

	e2e     *encryptionState
	storage *storageState
	hub     *hubState
}

var ssyk sasayakiState

func initSasayakiState(keyPair *disco.KeyPair, config *configuration) (*sasayakiState, error) {
	// hub needs a public key
	hubPublicKey, err := hex.DecodeString(config.HubPublicKey)
	if err != nil || len(hubPublicKey) != 32 {
		return nil, errors.New("ssyk: incorrect hub public key")
	}
	//
	ssyk := &sasayakiState{
		myAddress: keyPair.ExportPublicKey(),
		e2e:       initEncryptionState(keyPair),
		storage:   initStorageState(),
		hub:       initHubState(config.hubAddress, hubPublicKey),
	}
	return ssyk
}

// getNextMessage retrieves and decrypt a new message from the hub
// message order is ensured by the server, otherwise it will break the thread
// messages can also be contact requests, or contact acceptance
func (ss sasayakiState) getNextMessage() (*plaintextMsg, error) {
	storage.queryMutex.Lock()
	defer storage.queryMutex.Unlock()
	// obtain next message from hub
	encryptedMsg, err := hub.getNextMessage()
	if err != nil {
		return nil, err
	}

	// no address == no new message
	if encryptedMsg.GetFromAddress() == "" {
		return nil, errors.New("ssyk: no new messages")
	}
	// TODO: sanitize encryptedMsg? are addresses 32-byte hex?

	// checking if we're expecting a handshake message
	switch _, status := storage.getStateContact(bobAddress); status {
	case noContact: // first handshake message
		addContactFromReq(encryptedMsg)
		return nil, nil // TODO: what do we return here? (should we return an interface?)
	case waitingForAccept: // second handshake message
		finalizeContact(encryptedMsg)
		return nil, nil // TODO: what do we return here?
	case waitingToAccept: // TODO: should we really handle this case or let the rest fail?
		// TODO: if server doesn't delete message without our request, it's not going to work
		// TODO: idea: the getNextMessage request could also contain an ack for the previous
		return nil, nil
	case contactAdded:
		return handleNewMessage(encryptedMsg)
	default:
		panic("should not happen")
	}
}

func (ss sasayakiState) handleNewMessage(encryptedMsg *s.ResponseMessage) (*plaintextMsg, error) {
	// check fields
	if len(encryptedMsg.GetConvoId()) != 32 {
		return nil, errors.New("ssyk: message received malformed")
	}

	// new convo? create it
	if !storage.ConvoExist(encryptedMsg.GetConvoId()) {

		// get thread states for me -> bob
		_, t2, err := storage.getThreadRatchetStates(encryptedMsg.GetFromAddress())
		if err != nil {
			return err
		}
		// create convo message
		threadState, s1, s2 := e2e.createConvoFromMessage(t2, encryptedMsg)

		// create the conversation with the current thread ratchet value
		storage.createConvo(encryptedMsg.GetConvoId(), encryptedMsg.GetFromAddress(), "", s1, s2)

		// update the thread state
		storage.updateThreadRatchetStates(encryptedMsg.GetFromAddress(), nil, threadState)

		// decrypt the title
		titleMessage, err := e2e.decryptMessage(s1, encryptedMsg)
		if err != nil {
			return err
		}

		// update the title
		storage.updateTitle(titleMessage.ConvoId, titleMessage.FromAddress, titleMessage.Content)

		// TODO: nil means new convo???
		return nil, nil
	}

	// get session keys
	_, c2, err := storage.getSessionKeys(encryptedMsg.GetConvoId(), encryptedMsg.GetFromAddress())
	if err != nil {
		return nil, err
	}
	// remove encryption
	decryptedMessage, strobeState, err := e2e.decryptMessage(c2, encryptedMsg)
	if err != nil {
		return nil, err
	}
	// store new state
	storage.updateSessionKeys(encryptedMsg.GetConvoId(), encryptedMsg.GetFromAddress(), nil, strobeState)
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
// in the case of a new thread, convoId must be "" and the content must be the thread's title
func (ss sasayakiState) sendMessage(msg *plaintextMsg) (string, error) {
	storage.queryMutex.Lock()
	defer storage.queryMutex.Unlock()
	// is it a new thread?
	if msg.ConvoId == "" {
		// generate convoId
		var randomBytes [16]byte
		if _, err := rand.Read(randomBytes[:]); err != nil {
			panic(err)
		}
		msg.ConvoId = hex.EncodeToString(randomBytes[:])

		// get thread states for me -> bob
		t1, _, err := storage.getThreadRatchetStates(msg.ToAddress)
		if err != nil {
			return nil, err
		}
		// create new convo
		threadState, s1, s2, err := e2e.createNewConvo(t1, msg)
		if err != nil {
			return "", err
		}
		// update the thread state
		storage.updateThreadRatchetStates(msg.ToAddress, threadState, nil)
		// create the conversation with the current thread ratchet value and a random convoId
		storage.createConvo(msg.ConvoId, msg.ToAddress, msg.Content, s1, s2)

		// encrypt the title and return it
		encryptedMessage, s1, err := e2e.encryptMessage(s1, msg)
		if err != nil {
			return "", err
		}
		// store message

		// update strobeState // TODO: this should store msg as well
		storage.updateSessionKeys(msg.ConvoId, msg.ToAddress, s1, nil)
		panic("todo")

		// send to hub
		if err := hub.sendMessage(encryptedMessage); err != nil {
			return "", err
		}
	} else { // nope, it's just a message
		// get strobeState
		s1, _, err := storage.getSessionKeys(msg.ConvoId, msg.ToAddress)
		if err != nil {
			return nil, err
		}
		// add encryption
		encryptedMessage, strobeState, err := e2e.encryptMessage(s1, msg)
		if err != nil {
			return "", err
		}
		// update strobeState		// TODO: this should store the message as well
		storage.updateSessionKeys(msg.ConvoId, msg.ToAddress, strobeState, nil)
		panic("todo")
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

// addContact creates a contact request
// TODO: what happens when we do that?
// should it send a message coming from us? Or a meta msg from the hub?
// maybe if we receive a msg from someone we don't know, we can assume it is a request
func (ss sasayakiState) aliceAddContact(bobAddress, bobName string) error {
	storage.queryMutex.Lock()
	defer storage.queryMutex.Unlock()
	if len(bobAddress) != 64 {
		return errors.New("ssyk: contact's address is malformed")
	}

	// check that contact doesn't already have a state
	_, status := storage.getStateContact(bobAddress)
	if status != noContact {
		return nil, errors.New("ssyk: contact has already been added")
	}

	// unserialize key
	bobPubKey, err := hex.DecodeString(bobAddress)
	if err != nil {
		return nil, nil, errors.New("ssyk: contact's address is not hexadecimal")
	}

	// needed by libdisco
	bob := &disco.KeyPair{}
	copy(bob.PublicKey[:], bobPubKey)

	// get first handshake message
	firstHandshakeMessage, serializedHandshakeState, err := e2e.addContact(bob, bobName)
	if err != nil {
		return err
	}

	// store the new contact with the serialized handshake
	storage.addContact(bobAddress, bobName, serializedHandshakeState)

	// create message to send
	var randomBytes [16]byte
	if _, err := rand.Read(randomBytes[:]); err != nil {
		panic(err)
	}
	msgToSend := &s.Request_Message{
		ToAddress: bobAddress,
		ConvoId:   hex.EncodeToString(randomBytes[:]),
		Content:   hex.EncodeToString(firstHandshakeMessage),
	}

	// forward request to hub
	hub.sendMessage(msgToSend)

	//
	return nil
}

// receiveContactRequest is called when a contact request is being received.
// It stores the firstHandshakeMessage for later use
func (ss sasayakiState) bobReceiveContactRequest(aliceAddress string, firstHandshakeMessage []byte) error {
	storage.queryMutex.Lock()
	defer storage.queryMutex.Unlock()
	// check in storage if we are at this step in the handshake
	if _, status := storage.getStateContact(aliceAddress); status != noContact {
		return errors.New("ssyk: contact has already been added")
	}
	// store the thread states + state
	storage.addContactFromReq(aliceAddress, aliceName, firstHandshakeMessage)
	//
	return nil
}

// bobAcceptContact finalizes the handshake from the responder side
// TODO: we should be able to acceptContact even if we did it in the past (
// for example alice could do addContact / deleteContact / addContact
// TODO: should it return the contact id or something usable by the app?
func (ss sasayakiState) bobAcceptContact(aliceAddress, aliceName string) error {
	storage.queryMutex.Lock()
	defer storage.queryMutex.Unlock()
	// TODO: move all these checks in ssyk?
	if len(aliceAddress) != 64 {
		return errors.New("ssyk: contact's address is malformed")
	}

	// check in storage if we are at this step in the handshake
	firstHandshakeMessage, status := storage.getStateContact(aliceAddress)
	if status != waitingToAccept {
		return nil, errors.New("ssyk: contact is not being added properly")
	}
	// unserializekey
	alicePubKey, err := hex.DecodeString(aliceAddress)
	if err != nil {
		return nil, nil, nil, errors.New("ssyk: contact's address is not hexadecimal")
	}
	// needed by libdisco
	alice := &disco.KeyPair{}
	copy(alice.PublicKey[:], alicePubKey)
	// parse handshake message and continue handshake
	ts1, ts2, secondHandshakeMsg, err := e2e.acceptContact(alice, aliceName, firstHandshakeMessage)
	if err != nil {
		return err
	}

	// update contact with thread states
	storage.updateContact(aliceAddress, ts1, ts2)

	// forward second handshake message to hub
	panic("no hub support")

	return nil
}

// aliceAckAcceptContact finalizes the handshake from the initiator side
// TODO: we need to receive the information that bob has accepted our contact request
// how? via receipt of a "meta" message?
// how are friend requests sent anyway?
func (ss sasayakiState) aliceAckAcceptContact(bobAddress string, secondHandshakeMessage []byte) error {
	storage.queryMutex.Lock()
	defer storage.queryMutex.Unlock()

	// TODO: where do we verify that bobAddress is proper?

	// check in storage if we are at this step in the handshake
	serializedHandshakeState, status := storage.getStateContact(bobAddress)
	if status != waitingForAccept {
		return errors.New("ssyk: contact has not been added properly")
	}

	// finish handshake and get threadStates
	ts1, ts2, err := e2e.finishAddContact(serializedHandshakeState, secondHandshakeMessage)
	if err != nil {
		return err
	}

	// store the thread states
	storage.updateContact(bobAddress, ts1, ts2)

	// hub?
	panic("no hub support")

	return nil
}

// deleteContact is used to delete a contact from storage
func (ss sasayakiState) deleteContact(bobAddress string) error {
	storage.queryMutex.Lock()
	defer storage.queryMutex.Unlock()
	panic("not implemented")

	// TODO: no fwd to hub right?

	return storage.deleteContact(bobAddress)
}
