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
// The message type is used for plaintext messages, protobuffer types are used for encrypted messages
//
//
// notes:
// - id and convo id are unsigned 64bit integers. We increment them for every message/convo. note that
// we only care about the tuple {convoid, toAddress, fromAddress} which has very little chance of colliding

package main

import (
	"encoding/binary"
	"encoding/hex"
	"errors"

	s "github.com/mimoo/sasayaki/serialization"

	"github.com/mimoo/StrobeGo/strobe"
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
	toAuthenticate := make([]byte, 8+32+32)
	binary.BigEndian.PutUint64(toAuthenticate[:8], msg.ConvoId)
	bobPubKey, err := hex.DecodeString(msg.ToAddress)
	if err != nil {
		return nil, err
	}
	copy(toAuthenticate[8:], ssyk.keyPair.PublicKey[:])
	copy(toAuthenticate[40:], bobPubKey)
	// encrypt message
	ciphertext := s1.Send_AEAD([]byte(msg.Content), toAuthenticate)
	// store new state
	storage.updateSessionKeys(msg.ConvoId, msg.ToAddress, s1.Serialize(), nil)
	// create return value
	encryptedMessage := &s.Request_Message{
		ToAddress: msg.ToAddress,
		Id:        msg.Id,
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
	// TODO: what else should we authenticate here?
	toAuthenticate := make([]byte, 8+32+32)
	binary.BigEndian.PutUint64(toAuthenticate[:8], encryptedMsg.GetConvoId())
	bobPubKey, err := hex.DecodeString(encryptedMsg.GetFromAddress())
	if err != nil {
		return nil, err
	}
	copy(toAuthenticate[8:], bobPubKey)
	copy(toAuthenticate[40:], ssyk.keyPair.PublicKey[:])
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
		Id:          encryptedMsg.GetId(),
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

	// create the conversation with the current thread ratchet value
	convoId := ds.getLastConvoId() + 1
	storage.createConvo(convoId, msg.ToAddress, msg.Content, s1.Serialize(), s2.Serialize())

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

	// check if the convoId is valid
	convoId := ds.getLastConvoId() + 1       // what we see
	if convoId > encryptedMsg.GetConvoId() { // shouldn't be higher
		return errors.New("ssyk: convoId seems to be stuck in the past")
		// TODO: this should be a fatal error with the contact
		// at this point the best course of action seems to be: send an alert msg to the contact telling him impossible to read
		// wait... this can happen if we create a thread without having collected current threads...
		// maybe msg being sent in one direction should have a prefix? mmmm....
		// better, conversations have a direction
		// so a conversation is a tuple {myAddress, BobAddress, convoId, direction}
		// no... a 128-bit random value is better (GUID)
	}

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
