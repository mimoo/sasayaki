//
// Storage Service
// ================
//
// This is using a pretty simple sqlite database (see below for the schema)
//
//
//

package main

import (
	"database/sql"
	"errors"
	"path/filepath"
	"sync"
)

type databaseState struct {
	db *sql.DB

	queryMutex sync.Mutex // one sql query at a time
}

var storage databaseState

// TODO: protect database with encryption under our passphrase
func initDatabaseManager() {
	location := filepath.Join(sasayakiFolder(), "database.db")
	var err error
	storage.db, err = sql.Open("sqlite3", location)
	if err != nil {
		panic(err)
	}

	// The local database.
	//
	// Note that `contacts.state` requires a bit more explanation. It contains either:
	// - [0|blob] : we sent a contact request, blob is the serialized handshakeState
	// - [1|blob] : we received a contact request, blob is the received handshake message
	// - [2|empty] : we are done with the handshake, blob is empty
	//
	createStatement := `
	CREATE TABLE IF NOT EXISTS contacts (
		id INTEGER PRIMARY KEY AUTOINCREMENT, -- unique integer per account
		publickey TEXT NOT NULL UNIQUE, 			-- public key of the contact
		date TIMESTAMP, 											-- date added
		name TEXT, 														-- name chosen by us (often given by organization)
		state BLOB, 													-- the serialized handshake (see comment above for more information)
		c1 BLOB, 															-- serialized strobe state to create threads ->
		c2 BLOB 															-- serialized strobe state to create threads <-
	);
	CREATE TABLE IF NOT EXISTS verifications (
		id INTEGER PRIMARY KEY AUTOINCREMENT, -- unique integer per verification
		publickey TEXT NOT NULL, 							-- the public key of the verified account
		who TEXT NOT NULL, 										-- who is verifying the account
		date TIMESTAMP, 											-- when this verification was done
		how TEXT, 														-- how this verification was done (facebook, twitter, irl, etc.)
		name TEXT, 														-- the name used by the verifier to identify the public key
		signature TEXT NOT NULL 							-- the actual public key
	);
	CREATE TABLE IF NOT EXISTS conversations (
		id TEXT NOT NULL, 										-- a 16-byte random value? TODO: outch? collisions?
		publickey TEXT NOT NULL , 						-- the public key of the other peer
		title TEXT, 													-- the title of the thread
		date_creation TIMESTAMP, 							-- the date the thread was created
		date_last_message TIMESTAMP, 					-- the date the last message was sent/received
		c1 BLOB, 															-- the serialized strobe state to send messages
		c2 BLOB 															-- the serialized strobe state to receive messages
	);
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT, -- 
		conversation_id TEXT NOT NULL , 			-- the conversation the message is part of
		date TIMESTAMP, 											-- time the message was sent/received
		senderIsMe BOOLEAN, 									-- 0: I sent the message, 1: I received the message
		message BLOB 													-- the actual message 
	);
	`
	if _, err := storage.db.Exec(createStatement); err != nil {
		panic(err)
	}

	// defer db.Close() // we never close the db
}

func (storage *databaseState) getMessages() {
	storage.queryMutex.Lock()
	defer storage.queryMutex.Unlock()

	selectStatement := "SELECT * FROM conversations;"
	_, err := storage.db.Exec(selectStatement)
	if err != nil {
		panic(err)
	}
}

func (storage *databaseState) getThreadRatchetStates(bobAddress string) ([]byte, []byte, error) {
	//
	storage.queryMutex.Lock()
	defer storage.queryMutex.Unlock()
	// query
	stmt, err := storage.db.Prepare("SELECT state, c1, c2 FROM contacts WHERE publickey = ?;")
	if err != nil {
		return nil, nil, err
	}
	rows, err := stmt.Query(bobAddress)
	if err != nil {
		return nil, nil, err
	}
	// result
	if !rows.Next() {
		return nil, nil, errors.New("ssyk: the contact is not ready for conversations yet")
	}
	var state, c1, c2 []byte
	err = rows.Scan(state, c1, c2)
	if err != nil {
		return nil, nil, err
	}
	// check contact state first
	if len(state) != 1 || state[0] != 1 {
		return nil, nil, errors.New("ssyk: the contact is not ready for conversations yet")
	}
	// return ratchet states
	return c1, c2, nil
}

// getSessionKeys finds the session keys for chatting with Bob {convoId, BobAddress}
//
// if it doesn't find session keys for a  tuple
func (storage *databaseState) getSessionKeys(convoId, bobAddress string) ([]byte, []byte, error) {
	storage.queryMutex.Lock()
	defer storage.queryMutex.Unlock()

	stmt, err := storage.db.Prepare("SELECT c1, c2 FROM conversations WHERE id=? AND publickey=?;")
	if err != nil {
		panic(err)
	}
	rows, err := stmt.Query(convoId, bobAddress)
	if err != nil {
		panic(err)
	}
	if !rows.Next() {
		return nil, nil, errors.New("ssyk: the contact is not ready for conversations yet")
	}
	var c1, c2 []byte
	err = rows.Scan(c1, c2)
	if err != nil {
		return nil, nil, err
	}

	return c1, c2, nil
}

// updateSessionKeys will crash if the database query doesn't work
func (storage *databaseState) updateSessionKeys(convoId, bobAddress string, c1, c2 []byte) {
	storage.queryMutex.Lock()
	defer storage.queryMutex.Unlock()

	if c1 == nil && c2 == nil {
		panic("ssyk: at least one session key must be defined in order to call updateSessionKeys")
	}
	// c1 by default
	sessionkey := c1
	query := "UPDATE conversations SET c1=? WHERE id=? AND publickey=?;"
	if c1 == nil {
		sessionkey = c2
		query = "UPDATE conversations SET c2=? WHERE id=? AND publickey=?;"
	}
	stmt, err := storage.db.Prepare(query)
	if err != nil {
		panic(err)
	}
	_, err = stmt.Exec(sessionkey, convoId, bobAddress)
	if err != nil {
		panic(err)
	}
}

func (storage *databaseState) createConvo(convoId, bobAddress, title string, sessionkey1, sessionkey2 []byte) {
	// lock
	storage.queryMutex.Lock()
	defer storage.queryMutex.Unlock()

	// (id TEXT, publickey TEXT, title TEXT, date_creation TIMESTAMP, date_last_message TIMESTAMP, c1 BLOB, c2 BLOB);
	stmt, err := storage.db.Prepare("INSERT INTO conversations VALUES(?, ?, ?, DATETIME('now'), DATETIME('now'), ?, ?);")
	if err != nil {
		panic(err)
	}
	_, err = stmt.Exec(convoId, bobAddress, title, sessionkey1, sessionkey2)
	if err != nil {
		panic(err)
	}
}

// updateThreadRatchetStates takes two serialized thread states and update the bob's contact with them
func (storage *databaseState) updateThreadRatchetStates(bobAddress string, ts1, ts2 []byte) {
	storage.queryMutex.Lock()
	defer storage.queryMutex.Unlock()
	//
	if ts1 == nil && ts2 == nil {
		panic("ssyk: at least one session key must be defined in order to call updateSessionKeys")
	}
	// c1 by default
	threadState := ts1
	query := "UPDATE contacts SET c1=? WHERE publickey=?;"
	if threadState == nil {
		threadState = ts2
		query = "UPDATE contacts SET c2=? WHERE publickey=?;"
	}
	stmt, err := storage.db.Prepare(query)
	if err != nil {
		panic(err)
	}
	_, err = stmt.Exec(threadState, bobAddress)
	if err != nil {
		panic(err)
	}
}

// TODO: do I need a pointervalue to be able to Lock/Unlock on the mutex?
func (storage *databaseState) updateTitle(convoId, bobAddress, title string) {
	storage.queryMutex.Lock()
	defer storage.queryMutex.Unlock()
	//
	stmt, err := storage.db.Prepare("UPDATE conversations SET title=? WHERE id=? AND publickey=?;")
	if err != nil {
		panic(err)
	}

	_, err = stmt.Exec(title, convoId, bobAddress)
	if err != nil {
		panic(err)
	}
}

func (storage *databaseState) storeMessage(msg *plaintextMsg) uint64 {
	storage.queryMutex.Lock()
	defer storage.queryMutex.Unlock()
	// who sent it?
	senderIsMe := true
	if msg.FromAddress != ssyk.myAddress {
		senderIsMe = false
	}
	// messages (id INTEGER PRIMARY KEY AUTOINCREMENT, conversation_id INTEGER, date TIMESTAMP, senderIsMe TEXT, message BLOB)
	stmt, err := storage.db.Prepare("INSERT INTO messages VALUES(NULL, ?, DATETIME('now'), ?, ?);")
	if err != nil {
		panic(err)
	}
	res, err := stmt.Exec(msg.ConvoId, senderIsMe, msg.Content)
	if err != nil {
		panic(err)
	}
	// return id created
	id, _ := res.LastInsertId()
	return uint64(id)
}

func (storage *databaseState) ConvoExist(convoId string) bool {
	storage.queryMutex.Lock()
	defer storage.queryMutex.Unlock()

	stmt, err := storage.db.Prepare("SELECT id FROM conversations WHERE id=? LIMIT 1;")
	if err != nil {
		panic(err)
	}
	rows, err := stmt.Query(convoId)
	if err != nil {
		panic(err) // TODO: what can panic here?
	}
	return rows.Next()
}

type contactState uint8

const (
	noContact        contactState = iota // the contact hasn't been added yet
	waitingForAccept                     // the contact has been added, waiting for 2nd handshake message
	waitingToAccept                      // the contact has been added, waiting to send 2nd handshake message
	contactAdded                         // the contact has been successfuly added
)

// getStateContact returns nil if no contact has been added yet,
// otherwise it returns the state (xxxxxx=waiting for answer, 1=all good)
func (storage *databaseState) getStateContact(bobAddress string) ([]byte, contactState) {
	storage.queryMutex.Lock()
	defer storage.queryMutex.Unlock()
	//
	stmt, err := storage.db.Prepare("SELECT state FROM contacts WHERE publickey=?;")
	if err != nil {
		panic(err)
	}
	rows, err := stmt.Query(bobAddress)
	if err != nil {
		panic(err) // TODO: what can panic here?
	}
	if !rows.Next() {
		return nil, noContact
	}
	var state []byte
	err = rows.Scan(state)
	if err != nil {
		panic(err)
	}

// - [0|blob] : we sent a contact request, blob is the serialized handshakeState
// - [1|blob] : we received a contact request, blob is the received handshake message
// - [2|empty] : we are done with the handshake, blob is empty
	if state[0] == 0 {
		return state[1:], waitingForAccept
	} else if state[0] == 1 {
		return state[1:], waitingToAccept
	} 

	return nil, contactAdded
}

// addContact is used when adding a contact for the very first time 
// (which is supposed to send a handshake message)
// Note that `contacts.state` requires a bit more explanation. It contains either:
// - [0|blob] : we sent a contact request, blob is the serialized handshakeState
// - [1|blob] : we received a contact request, blob is the received handshake message
// - [2|empty] : we are done with the handshake, blob is empty
func (storage *databaseState) addContact(bobAddress, bobName string, serializedHandshakeState []byte) {
	storage.queryMutex.Lock()
	defer storage.queryMutex.Unlock()

	// contacts (id INTEGER PRIMARY KEY AUTOINCREMENT, publickey TEXT, date TIMESTAMP, name TEXT, state BLOB, c1 BLOB, c2 BLOB);
	stmt, err := storage.db.Prepare("INSERT INTO contacts VALUES(NULL, ?, DATETIME('now'), ?, ?, NULL, NULL);")
	if err != nil {
		panic(err)
	}
	_, err = stmt.Exec(bobAddress, bobName, append([]byte{0, serializedHandshakeState...})
	if err != nil {
		panic(err)
	}
}

// addContactFromReq is used to add a new contact entry from a received contact request 
// this function assumes that there is not already a contact for this entry
func (storage *databaseState) addContactFromReq(aliceAddress string, firstHandshakeMessage []byte) {
	// lock db
	storage.queryMutex.Lock()
	defer storage.queryMutex.Unlock()

	// 
	stmt, err := storage.db.Prepare("INSERT INTO contacts VALUES(NULL, ?, DATETIME('now'), NULL, ?, NULL, NULL);")
	if err != nil {
		panic(err)
	}
	_, err = stmt.Exec(aliceAddress, append([]byte{1, firstHandshakeMessage...})
	if err != nil {
		panic(err)
	}


}

// updateContact is used when finalized a handshake by both peers
// Note that `contacts.state` requires a bit more explanation. It contains either:
// - [0|blob] : we sent a contact request, blob is the serialized handshakeState
// - [1|blob] : we received a contact request, blob is the received handshake message
// - [2|empty] : we are done with the handshake, blob is empty
func (storage *databaseState) finalizeContact(bobAddress, ts1, ts2 []byte) error {
	storage.queryMutex.Lock()
	defer storage.queryMutex.Unlock()
	// contacts (id INTEGER PRIMARY KEY AUTOINCREMENT, publickey TEXT, date TIMESTAMP, name TEXT, state BLOB, c1 BLOB, c2 BLOB);
	stmt, err = storage.db.Prepare("UPDATE contacts SET state=?, c1=?, c2=? WHERE publickey=?;")
	if err != nil {
		panic(err)
	}
	res, err = stmt.Exec([]byte{2}, ts1, ts2, bobAddress)
	if err != nil {
		panic(err)
	}
	affect, err = res.RowsAffected()
	if err != nil {
		panic(err)
	}
	if len(affect) != 1 {
		return errors.New("ssyk: contact does not exist")
	}
	//
	return nil
}

func updateContactName(bobAddress, bobName string) error {
	storage.queryMutex.Lock()
	defer storage.queryMutex.Unlock()
	// contacts (id INTEGER PRIMARY KEY AUTOINCREMENT, publickey TEXT, date TIMESTAMP, name TEXT, state BLOB, c1 BLOB, c2 BLOB);
	stmt, err = storage.db.Prepare("UPDATE contacts SET name=? WHERE publickey=?;")
	if err != nil {
		panic(err)
	}
	res, err = stmt.Exec(bobName, bobAddress)
	if err != nil {
		panic(err)
	}
	affect, err = res.RowsAffected()
	if err != nil {
		panic(err)
	}
	if len(affect) != 1 {
		return errors.New("ssyk: contact does not exist")
	}
	return nil
}

func (storage *databaseState) deleteContact(bobAddress string) error {
	storage.queryMutex.Lock()
	defer storage.queryMutex.Unlock()

	stmt, err = storage.db.Prepare("DELETE FROM contacts WHERE publickey=?;")
	if err != nil {
		panic(err)
	}
	res, err = stmt.Exec(bobAddress)
	if err != nil {
		panic(err)
	}
	affect, err = res.RowsAffected()
	if err != nil {
		panic(err)
	}
	if len(affect) != 1 {
		return errors.New("ssyk: contact does not exist")
	}
	return nil
}
