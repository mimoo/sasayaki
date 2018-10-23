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

type encryptionState struct {
	keyPair *disco.KeyPair
}

var e2e encryptionState

func initEncryptionState(keyPair *disco.KeyPair) *encryptionState {
	e2e := &encryptionState{
		keyPair: keyPair,
	}
	return e2e
}

//
// Messages
// ========
//

// encryptMessage takes a message type and returns a protobuf request containing
// the encrypted message
func (e2e encryptionState) encryptMessage(strobeState []byte, msg *plaintextMsg) (*s.Request_Message, []byte, error) {
	// check for arbitrary 1000 bytes of room for headers and protobuff structure
	if len(msg.Content) > 65535-1000 {
		return nil, nil, errors.New("ssyk: message to send is too large")
	}

	// recover strobe state
	s1 := strobe.RecoverState(strobeState)
	// data to authenticate [convoId(8), sendPubKey(32), recvPubKey(32)]
	// TODO: what else should we authenticate here?
	toAuthenticate := make([]byte, 16+32+32)
	convoId, err := hex.DecodeString(msg.ConvoId)
	if err != nil {
		return nil, nil, err
	}
	bobPubKey, err := hex.DecodeString(msg.ToAddress)
	if err != nil {
		return nil, nil, err
	}
	copy(toAuthenticate[0:16], convoId)
	copy(toAuthenticate[16:16+32], e2e.keyPair.PublicKey[:])
	copy(toAuthenticate[16+32:16+32+32], bobPubKey)
	// encrypt message
	ciphertext := s1.Send_AEAD([]byte(msg.Content), toAuthenticate)
	// create return value
	encryptedMessage := &s.Request_Message{
		ToAddress: msg.ToAddress,
		ConvoId:   msg.ConvoId,
		Content:   ciphertext,
	}
	// return ciphertext
	return encryptedMessage, s1.Serialize(), nil
}

// decryptMessage takes a protobuf encrypted responseMessage and returns the decrypted content
func (e2e encryptionState) decryptMessage(strobeState []byte, encryptedMsg *s.ResponseMessage) (*plaintextMsg, []byte, error) {
	// check
	if encryptedMsg.GetContent() == nil {
		return nil, nil, errors.New("ssyk: message received is incorrectly formed")
	}
	// data to authenticate [convoId(8), sendPubKey(32), recvPubKey(32)]
	toAuthenticate := make([]byte, 16+32+32)
	convoId, err := hex.DecodeString(encryptedMsg.GetConvoId())
	if err != nil {
		return nil, nil, err
	}
	bobPubKey, err := hex.DecodeString(encryptedMsg.GetFromAddress())
	if err != nil {
		return nil, nil, err
	}
	copy(toAuthenticate[0:16], convoId)
	copy(toAuthenticate[16:16+32], bobPubKey)
	copy(toAuthenticate[16+32:16+32+32], e2e.keyPair.PublicKey[:])

	// decrypt message
	s2 := strobe.RecoverState(strobeState)
	plaintext, ok := s2.Recv_AEAD(encryptedMsg.GetContent(), toAuthenticate)
	if !ok {
		return nil, nil, errors.New("ssyk: impossible to decrypt incoming message")
		// TODO: this should completely kill the thread
	}
	// return plaintext
	msg := &plaintextMsg{
		ConvoId:     encryptedMsg.GetConvoId(),
		FromAddress: encryptedMsg.GetFromAddress(),
		ToAddress:   e2e.keyPair.ExportPublicKey(),
		Content:     string(plaintext),
	}
	//
	return msg, s2.Serialize(), nil
}

// createNewConvo returns the new threadState (after ratcheting) and the two session keys created for the thread
func (e2e encryptionState) createNewConvo(threadState []byte, msg *plaintextMsg) (*s.Request_Message, []byte, []byte, []byte, error) {
	// recover state
	threadState := strobe.RecoverState(threadState)

	// create the session keys for the convo (following disco spec)
	s1 := threadState.Clone()
	s2 := threadState.Clone()

	s1.AD(true, []byte("initiator"))
	s1.RATCHET(32)

	s2.AD(true, []byte("responder"))
	s2.RATCHET(32)

	// ratchet the thread state (following disco spec)
	threadState.RATCHET(32)

	// encrypt the title and return it
	return e2e.encryptMessage(msg), threadState.Serialize(), s1.Serialize(), s2.Serialize(), nil
}

// from a threadState, create new session keys. Ratchets the state. returns all
func (e2e encryptionState) createConvoFromMessage(threadState []byte) ([]byte, []byte, []byte) {
	// recover state
	ts := strobe.RecoverState(threadState)

	// create the session keys for the convo (following disco spec)
	s1 := ts.Clone()
	s2 := ts.Clone()

	s1.AD(true, []byte("initiator"))
	s1.RATCHET(32)

	s2.AD(true, []byte("responder"))
	s2.RATCHET(32)

	// ratchet the thread state (following disco spec)
	ts.RATCHET(32)

	//
	return ts.Serialize(), s1.Serialize(), s2.Serialize()
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
func (e2e encryptionState) addContact(bobAddress, bobName string) ([]byte, []byte, error) {
	// TODO: prologue?
	// idea: [addContact|myAddress|bobAddress]
	prologue := []byte{}

	// unserialize key
	bobPubKey, err := hex.DecodeString(bobAddress)
	if err != nil {
		return nil, nil, errors.New("ssyk: contact's address is not hexadecimal")
	}
	// needed by libdisco
	bob := &disco.KeyPair{}
	copy(bob.PublicKey[:], bobPubKey)

	// Initialize Disco
	hs := disco.Initialize(disco.Noise_IK, true, prologue, e2e.keyPair, nil, bob, nil)

	// write the first message
	var msg []byte
	if _, _, err := hs.WriteMessage(nil, &msg); err != nil {
		panic(err)
	}

	//
	return msg, hs.Serialize(), nil
}

// acceptContact parses the first Noise handshake message -> e, es, s, ss
// then produces the second (and final) Noise handshake message <- e, ee, se
// this produces two strobe states that can be used to create threads between the two contacts
func (e2e encryptionState) acceptContact(aliceAddress, aliceName string, firstHandshakeMessage []byte) ([]byte, []byte, []byte, error) {
	// unserializekey
	alicePubKey, err := hex.DecodeString(aliceAddress)
	if err != nil {
		return nil, nil, nil, errors.New("ssyk: contact's address is not hexadecimal")
	}
	// needed by libdisco
	alice := &disco.KeyPair{}
	copy(alice.PublicKey[:], alicePubKey)

	// TODO: prologue
	prologue := []byte{}

	// initialize handshake state
	hs := disco.Initialize(disco.Noise_IK, false, prologue, e2e.keyPair, nil, alice, nil)
	if _, _, err := hs.ReadMessage(firstHandshakeMessage, nil); err != nil {
		return nil, nil, nil, err
	}

	// write the second handshake message
	var msg []byte
	ts2, ts1, err := hs.WriteMessage(nil, &msg) // reversed because we are the responder
	if err != nil {
		return nil, nil, nil, err
	}

	//
	return ts1.Serialize(), ts2.Serialize(), msg, nil
}

// finishHandshake parses the second (and final) Noise handshake message <- e, ee, se
func (e2e encryptionState) finishAddContact(serializedHandshakeState, secondHandshakeMessage []byte) ([]byte, []byte, error) {
	// unserialize handshake state
	hs := disco.RecoverState(serializedHandshakeState, nil, e2e.keyPair)

	// parse last message
	var payload []byte
	ts1, ts2, err := hs.ReadMessage(secondHandshakeMessage, &payload)
	if err != nil {
		return nil, nil, err
	}

	//
	return ts1.Serialize(), ts2.Serialize(), nil
}
