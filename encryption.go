//
// Encryption Manager
// ==================
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
//
//
// notes:
// - id and convo id are unsigned 64bit integers. We can generate them randomly, it shouldn't matter
// 	since we care about the tuple {convoid, toAddress, fromAddress} which has very little chance of colliding

package main

import (
	"encoding/binary"
	"encoding/hex"
	"errors"

	"github.com/mimoo/StrobeGo/strobe"
)

type encryptionManager struct {
	ds *databaseState
}

var em encryptionManager

func initEncryptionManager(ds *databaseState) {
	em.ds = ds
}

func (em encryptionManager) encryptMessage(convoId uint64, bobAddress string, content string) ([]byte, error) {
	// get session keys
	c1, _, err := ds.getSessionKeys(convoId, bobAddress)
	if err != nil {
		return nil, err
	}
	// recover strobe state
	s1 := strobe.RecoverState(c1)
	// data to authenticate [convoId(8), sendPubKey(32), recvPubKey(32)]
	// TODO: what else should we authenticate here?
	toAuthenticate := make([]byte, 8+32+32)
	binary.BigEndian.PutUint64(toAuthenticate[:8], convoId)
	bobPubKey, err := hex.DecodeString(bobAddress)
	if err != nil {
		return nil, err
	}
	copy(toAuthenticate[8:], ss.keyPair.PublicKey[:])
	copy(toAuthenticate[40:], bobPubKey)
	// encrypt message
	ciphertext := s1.Send_AEAD([]byte(content), toAuthenticate)
	// store new state
	ds.updateSessionKeys(convoId, bobAddress, s1.Serialize(), nil)
	// return ciphertext
	return ciphertext, nil
}

func (em encryptionManager) decryptMessage(convoId uint64, bobAddress string, ciphertext []byte) ([]byte, error) {
	// get session keys
	_, c2, err := ds.getSessionKeys(convoId, bobAddress)
	if err != nil {
		return nil, err
	}
	// data to authenticate [convoId(8), sendPubKey(32), recvPubKey(32)]
	// TODO: what else should we authenticate here?
	toAuthenticate := make([]byte, 8+32+32)
	binary.BigEndian.PutUint64(toAuthenticate[:8], convoId)
	bobPubKey, err := hex.DecodeString(bobAddress)
	if err != nil {
		return nil, err
	}
	copy(toAuthenticate[8:], bobPubKey)
	copy(toAuthenticate[40:], ss.keyPair.PublicKey[:])
	// decrypt message
	s2 := strobe.RecoverState(c2)
	plaintext, ok := s2.Recv_AEAD(ciphertext, toAuthenticate)
	if !ok {
		return nil, errors.New("ssyk: impossible to decrypt incoming message")
		// TODO: this should completely kill the thread
	}
	// store new state
	ds.updateSessionKeys(convoId, bobAddress, nil, s2.Serialize())
	// return plaintext, new state (to be stored)
	return plaintext, nil
}

func (em encryptionManager) createNewConvo(bobAddress string, title string) ([]byte, error) {
	// get thread states for me -> bob
	t1, _, err := ds.getThreadRatchetStates(bobAddress)
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

	// create the conversation with the current thread ratchet value
	convoId := ds.createConvo(bobAddress, title, s1.Serialize(), s2.Serialize())

	// ratchet the thread state (following disco spec)
	threadState.RATCHET(32)

	// update the thread state
	ds.updateThreadRatchetStates(bobAddress, threadState.Serialize(), nil)

	// encrypt the title and return it
	return em.encryptMessage(convoId, bobAddress, title)
}

func (em encryptionManager) createConvoFromMessage(bobAddress string, ciphertext []byte) error {
	// get thread states for me -> bob
	_, t2, err := ds.getThreadRatchetStates(bobAddress)
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
	convoId := ds.createConvo(bobAddress, "", s1.Serialize(), s2.Serialize())

	// decrypt the title
	title, err := em.decryptMessage(convoId, bobAddress, ciphertext)
	if err != nil {
		return err
	}

	// update the title
	ds.updateTitle(convoId, bobAddress, string(title))

	// ratchet the thread state (following disco spec)
	threadState.RATCHET(32)

	// update the thread state
	ds.updateThreadRatchetStates(bobAddress, nil, threadState.Serialize())

	//
	return nil
}
