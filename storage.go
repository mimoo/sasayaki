//
// Database Manager
// ================
//
// Contacts
// - id
// - publickey: of the account
// - date: metadata
// - name: hector
// - state: 0=none, xxxxxx=waiting for answer, 1=c1 and c2 good to be used
// - c1: state for creating threads ->
// - c2: state for creating threads <-
//
// Verifications
// - id
// - publickey: of the verified account
// - who: publickey of verifier
// - date: metadata
// - how: via facebook
// - name: hector
// - signature: signature from "who" over "'verification' | date | publickey | len_name | name | len_how | how"
//
// Conversations
// - id: we can have different convos with the same person (like email)
// - publickey: of the account
// - title: metadata
// - date_creation: metadata
// - date_last_message: metadata
// - c1: state after the last message ->
// - c2: state after the last message <-
//
// Messages
// - id
// - conversation_id
// - date: metadata
// - sender: me or him
// - message: actual content

package main

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"
)

type databaseState struct {
	db *sql.DB

	queryMutex sync.Mutex // one sql query at a time
}

var storage databaseState

func initDatabaseManager() {
	location := filepath.Join(sasayakiFolder(), "database.db")
	var err error
	storage.db, err = sql.Open("sqlite3", location)
	if err != nil {
		panic(err)
	}

	createStatement := `
	CREATE TABLE IF NOT EXISTS contacts (id INTEGER PRIMARY KEY AUTOINCREMENT, publickey TEXT, date TIMESTAMP, name TEXT, state BLOB, c1 BLOB, c2 BLOB);
	CREATE TABLE IF NOT EXISTS verifications (id INTEGER PRIMARY KEY AUTOINCREMENT, publickey TEXT, who TEXT, date TIMESTAMP, how TEXT, name TEXT, signature TEXT);
	CREATE TABLE IF NOT EXISTS conversations (id INTEGER PRIMARY KEY AUTOINCREMENT, publickey TEXT, title TEXT, date_creation TIMESTAMP, date_last_message TIMESTAMP, c1 BLOB, c2 BLOB);
	CREATE TABLE IF NOT EXISTS messages (id INTEGER PRIMARY KEY AUTOINCREMENT, conversation_id INTEGER, date TIMESTAMP, sender TEXT, message BLOB);
	`
	_, err = storage.db.Exec(createStatement)
	if err != nil {
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
	rows.Next()
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
func (storage *databaseState) getSessionKeys(convoId uint64, bobAddress string) ([]byte, []byte, error) {
	storage.queryMutex.Lock()
	defer storage.queryMutex.Unlock()

	stmt, err := storage.db.Prepare("SELECT state, c1, c2 FROM conversations WHERE id=? AND publickey=?")
	if err != nil {
		panic(err)
	}
	rows, err := stmt.Query(convoId, bobAddress)
	if err != nil {
		panic(err)
	}
	rows.Next()
	var state, c1, c2 []byte
	err = rows.Scan(state, c1, c2)
	if err != nil {
		return nil, nil, err
	}
	if len(state) != 1 || state[0] != 1 {
		return nil, nil, errors.New("ssyk: the contact is not ready for conversations yet")
	}

	return c1, c2, nil
}

// updateSessionKeys will crash if the database query doesn't work
func (storage *databaseState) updateSessionKeys(convoId uint64, bobAddress string, c1, c2 []byte) {
	storage.queryMutex.Lock()
	defer storage.queryMutex.Unlock()

	if c1 == nil && c2 == nil {
		panic("ssyk: at least one session key must be defined in order to call updateSessionKeys")
	}
	// c1 by default
	sessionkey := c1
	query := "UPDATE conversations SET c1=? WHERE id=? AND publickey=?"
	if c1 == nil {
		sessionkey = c2
		query = "UPDATE conversations SET c2=? WHERE id=? AND publickey=?"
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

func (storage *databaseState) getLastConvoId() uint64 {
	storage.queryMutex.Lock()
	defer storage.queryMutex.Unlock()

	selectStatement := "SELECT id FROM conversations ORDER BY id DESC LIMIT 1;"
	rows, err := storage.db.Query(selectStatement)
	if err != nil {
		panic(err)
	}

	rows.Next()
	var lastConvoId uint64
	err = rows.Scan(lastConvoId)
	if err != nil {
		fmt.Println("no convo yet? delete this msg if it worked")
		return 0
	}

	return lastConvoId
}

func (storage *databaseState) createConvo(bobAddress, title string, sessionkey1, sessionkey2 []byte) uint64 {
	storage.queryMutex.Lock()
	defer storage.queryMutex.Unlock()

	// (id INTEGER PRIMARY KEY AUTOINCREMENT, publickey TEXT, title TEXT, date_creation TIMESTAMP, date_last_message TIMESTAMP, c1 BLOB, c2 BLOB);
	stmt, err := storage.db.Prepare("INSERT INTO conversations VALUES(NULL, ?, ?, ?, ?, ?, ?);")
	if err != nil {
		panic(err)
	}
	res, err := stmt.Exec(bobAddress, title, bobAddress, title, time.Now(), time.Now(), sessionkey1, sessionkey2)
	if err != nil {
		panic(err)
	}
	// return convoId created
	convoId, _ := res.LastInsertId()
	return uint64(convoId)
}

// updateThreadRatchetStates takes two serialized thread states and update the bob's contact with them
func (storage *databaseState) updateThreadRatchetStates(bobAddress string, ts1, ts2 []byte) {
	storage.queryMutex.Lock()
	defer storage.queryMutex.Unlock()

	if ts1 == nil && ts2 == nil {
		panic("ssyk: at least one session key must be defined in order to call updateSessionKeys")
	}
	// c1 by default
	threadState := ts1
	query := "UPDATE contacts SET c1=? WHERE publickey=?"
	if threadState == nil {
		threadState = ts2
		query = "UPDATE contacts SET c2=? WHERE publickey=?"
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

func (storage *databaseState) updateTitle(convoId uint64, bobAddress, title string) {
	storage.queryMutex.Lock()
	defer storage.queryMutex.Unlock()

	stmt, err := storage.db.Prepare("UPDATE conversations SET title=? WHERE id=? AND publickey=?")
	if err != nil {
		panic(err)
	}

	_, err = stmt.Exec(title, convoId, bobAddress)
	if err != nil {
		panic(err)
	}
}
