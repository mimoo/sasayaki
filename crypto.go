//
// End-to-End Encryption Service
// =============================
//
// Each contact is either ready, or not, for conversations (depending on if they finished or not their handshake)
//
// Each on-going conversation has two states:
//
// * Alice -> Bob state
// * Bob -> Alice state
//
// When encrypting or decrypting a message, one of these states needs to be fetched from the database
//
// The message type is used for plaintext messages, protobuffer types are used for encrypted messages
//
//
// notes:
// - id and convo id are unsigned 64bit integers. We increment them for every message/convo. note that
// we only care about the tuple {convoid, toAddress, fromAddress} which has very little chance of colliding

package main

import (
	"encoding/hex"
	"errors"

	s "github.com/mimoo/sasayaki/serialization"

	"github.com/mimoo/StrobeGo/strobe"
	disco "github.com/mimoo/disco/libdisco"
)

type encryptionManager struct {
	//	storage *databaseState
}

var e2e encryptionManager

/*
func initEncryptionManager(storage *databaseState) {
	e2e.storage = storage
}
*/

//
// Messages
// ========
//

// encryptMessage takes a message type and returns a protobuf request containing
// the encrypted message
func (e2e encryptionManager) encryptMessage(msg *plaintextMsg) (*s.Request_Message, error) {
	// check for arbitrary 1000 bytes of room for headers and protobuff structure
	if len(msg.Content) > 65535-1000 {
		return nil, errors.New("ssyk: message to send is too large")
	}
	// get session keys
	c1, _, err := storage.getSessionKeys(msg.ConvoId, msg.ToAddress)
	if err != nil {
		return nil, err
	}
	// recover strobe state
	s1 := strobe.RecoverState(c1)
	// data to authenticate [convoId(8), sendPubKey(32), recvPubKey(32)]
	// TODO: what else should we authenticate here?
	toAuthenticate := make([]byte, 16+32+32)
	convoId, err := hex.DecodeString(msg.ConvoId)
	if err != nil {
		return nil, err
	}
	bobPubKey, err := hex.DecodeString(msg.ToAddress)
	if err != nil {
		return nil, err
	}
	copy(toAuthenticate[0:16], convoId)
	copy(toAuthenticate[16:16+32], ssyk.keyPair.PublicKey[:])
	copy(toAuthenticate[16+32:16+32+32], bobPubKey)
	// encrypt message
	ciphertext := s1.Send_AEAD([]byte(msg.Content), toAuthenticate)
	// store new state
	storage.updateSessionKeys(msg.ConvoId, msg.ToAddress, s1.Serialize(), nil)
	// create return value
	encryptedMessage := &s.Request_Message{
		ToAddress: msg.ToAddress,
		ConvoId:   msg.ConvoId,
		Content:   ciphertext,
	}
	// return ciphertext
	return encryptedMessage, nil
}

// decryptMessage takes a protobuf encrypted responseMessage and returns the decrypted content
func (e2e encryptionManager) decryptMessage(encryptedMsg *s.ResponseMessage) (*plaintextMsg, error) {
	// check
	if encryptedMsg.GetContent() == nil {
		return nil, errors.New("ssyk: message received is incorrectly formed")
	}
	// get session keys
	_, c2, err := storage.getSessionKeys(encryptedMsg.GetConvoId(), encryptedMsg.GetFromAddress())
	if err != nil {
		return nil, err
	}
	// data to authenticate [convoId(8), sendPubKey(32), recvPubKey(32)]
	toAuthenticate := make([]byte, 16+32+32)
	convoId, err := hex.DecodeString(encryptedMsg.GetConvoId())
	if err != nil {
		return nil, err
	}
	bobPubKey, err := hex.DecodeString(encryptedMsg.GetFromAddress())
	if err != nil {
		return nil, err
	}
	copy(toAuthenticate[0:16], convoId)
	copy(toAuthenticate[16:16+32], bobPubKey)
	copy(toAuthenticate[16+32:16+32+32], ssyk.keyPair.PublicKey[:])

	// decrypt message
	s2 := strobe.RecoverState(c2)
	plaintext, ok := s2.Recv_AEAD(encryptedMsg.GetContent(), toAuthenticate)
	if !ok {
		return nil, errors.New("ssyk: impossible to decrypt incoming message")
		// TODO: this should completely kill the thread
	}
	// store new state
	storage.updateSessionKeys(encryptedMsg.GetConvoId(), encryptedMsg.GetFromAddress(), nil, s2.Serialize())
	// return plaintext
	msg := &plaintextMsg{
		ConvoId:     encryptedMsg.GetConvoId(),
		FromAddress: encryptedMsg.GetFromAddress(),
		ToAddress:   ssyk.myAddress,
		Content:     string(plaintext),
	}
	//
	return msg, nil
}

func (e2e encryptionManager) createNewConvo(msg *plaintextMsg) (*s.Request_Message, error) {
	// get thread states for me -> bob
	t1, _, err := storage.getThreadRatchetStates(msg.ToAddress)
	if err != nil {
		return nil, err
	}

	// recover state
	threadState := strobe.RecoverState(t1)

	// create the session keys for the convo (following disco spec)
	s1 := threadState.Clone()
	s2 := threadState.Clone()

	s1.AD(true, []byte("initiator"))
	s1.RATCHET(32)

	s2.AD(true, []byte("responder"))
	s2.RATCHET(32)

	// create the conversation with the current thread ratchet value and a random convoId
	storage.createConvo(msg.ConvoId, msg.ToAddress, msg.Content, s1.Serialize(), s2.Serialize())

	// ratchet the thread state (following disco spec)
	threadState.RATCHET(32)

	// update the thread state
	storage.updateThreadRatchetStates(msg.ToAddress, threadState.Serialize(), nil)

	// encrypt the title and return it
	return e2e.encryptMessage(msg)
}

func (e2e encryptionManager) createConvoFromMessage(encryptedMsg *s.ResponseMessage) error {
	// get thread states for me -> bob
	_, t2, err := storage.getThreadRatchetStates(encryptedMsg.GetFromAddress())
	if err != nil {
		return err
	}

	// recover state
	threadState := strobe.RecoverState(t2)

	// create the session keys for the convo (following disco spec)
	s1 := threadState.Clone()
	s2 := threadState.Clone()

	s1.AD(true, []byte("initiator"))
	s1.RATCHET(32)

	s2.AD(true, []byte("responder"))
	s2.RATCHET(32)

	// create the conversation with the current thread ratchet value
	storage.createConvo(encryptedMsg.GetConvoId(), encryptedMsg.GetFromAddress(), "", s1.Serialize(), s2.Serialize())

	// decrypt the title
	titleMessage, err := e2e.decryptMessage(encryptedMsg)
	if err != nil {
		return err
	}

	// update the title
	storage.updateTitle(titleMessage.ConvoId, titleMessage.FromAddress, titleMessage.Content)

	// ratchet the thread state (following disco spec)
	threadState.RATCHET(32)

	// update the thread state
	storage.updateThreadRatchetStates(encryptedMsg.GetFromAddress(), nil, threadState.Serialize())

	//
	return nil
}

//
// Contact Management
// ==================
//
//     IK:
//      <- s
//      ...
//      -> e, es, s, ss
//      <- e, ee, se
//

// addContact produces the first handshake message -> e, es, s, ss
// note that if this has already been called, it cannot be called again
// to re-add a contact, it must first be deleted
func (e2e encryptionManager) addContact(bobAddress, bobName string) ([]byte, error) {
	// check that contact doesn't already have a state
	_, status := storage.getStateContact(bobAddress)
	if status == waitingForAccept {
		return nil, errors.New("ssyk: contact has already been added")
	}
	if status == contactAdded {
		return nil, errors.New("ssyk: contact has already been added successfuly")
	}

	// TODO: prologue?
	// idea: [addContact|myAddress|bobAddress]
	prologue := []byte{}

	// unserialize key
	bobPubKey, err := hex.DecodeString(bobAddress)
	if err != nil {
		return nil, errors.New("ssyk: contact's address is not hexadecimal")
	}
	// needed by libdisco
	bob := &disco.KeyPair{}
	copy(bob.PublicKey[:], bobPubKey)

	// Initialize Disco
	hs := disco.Initialize(disco.Noise_IK, true, prologue, ssyk.keyPair, nil, bob, nil)

	// write the first message
	var msg []byte
	if _, _, err := hs.WriteMessage(nil, &msg); err != nil {
		panic(err)
	}

	// store the serialized state
	storage.addContact(bobAddress, bobName, hs.Serialize())

	//
	return msg, nil
}

// acceptContact parses the first Noise handshake message -> e, es, s, ss
// then produces the second (and final) Noise handshake message <- e, ee, se
// this produces two strobe states that can be used to create threads between the two contacts
func (e2e encryptionManager) acceptContact(aliceAddress, aliceName string, firstHandshakeMessage []byte) ([]byte, error) {
	// check in storage if we are at this step in the handshake
	_, status := storage.getStateContact(aliceAddress)
	//	if contact == waitingForAccept // it's possible that we've added them as well, ignore it
	if status == contactAdded {
		return nil, errors.New("ssyk: contact has already been added successfuly")
	}

	// unserializekey
	alicePubKey, err := hex.DecodeString(aliceAddress)
	if err != nil {
		return nil, errors.New("ssyk: contact's address is not hexadecimal")
	}
	// needed by libdisco
	alice := &disco.KeyPair{}
	copy(alice.PublicKey[:], alicePubKey)

	// TODO: prologue
	prologue := []byte{}

	// initialize handshake state
	hs := disco.Initialize(disco.Noise_IK, false, prologue, ssyk.keyPair, nil, alice, nil)
	if _, _, err := hs.ReadMessage(firstHandshakeMessage, nil); err != nil {
		return nil, err
	}

	// write the second handshake message
	var msg []byte
	ts2, ts1, err := hs.WriteMessage(nil, &msg) // reversed because we are the responder
	if err != nil {
		return nil, err
	}

	// store the thread states + state
	storage.addContact(aliceAddress, aliceName, nil)
	storage.updateContact(aliceAddress, ts1.Serialize(), ts2.Serialize())

	//
	return msg, nil
}

// finishHandshake parses the second (and final) Noise handshake message <- e, ee, se
func (e2e encryptionManager) finishHandshake(bobAddress string, secondHandshakeMessage []byte) error {
	// check in storage if we are at this step in the handshake
	serializedHandshakeState, status := storage.getStateContact(bobAddress)
	if status == noContact {
		return errors.New("ssyk: contact hasn't been added properly")
	}
	if status == contactAdded {
		return errors.New("ssyk: contact has already been added successfuly")
	}

	// unserialize handshake state
	hs := disco.RecoverState(serializedHandshakeState, nil, ssyk.keyPair)

	// necessary for libdisco
	bobPubKey, err := hex.DecodeString(bobAddress)
	if err != nil {
		return errors.New("ssyk: contact's address is not hexadecimal")
	}
	bob := &disco.KeyPair{}
	copy(bob.PublicKey[:], bobPubKey)

	// parse last message
	var payload []byte
	ts1, ts2, err := hs.ReadMessage(secondHandshakeMessage, &payload)
	if err != nil {
		return err
	}

	// store the thread states + state
	storage.updateContact(bobAddress, ts1.Serialize(), ts2.Serialize())

	//
	return nil
}
