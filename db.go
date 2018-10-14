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

var ds databaseState

func initDatabaseManager() {
	location := filepath.Join(sasayakiFolder(), "database.db")
	var err error
	ds.db, err = sql.Open("sqlite3", location)
	if err != nil {
		panic(err)
	}

	createStatement := `
	CREATE TABLE IF NOT EXISTS contacts (id INTEGER PRIMARY KEY AUTOINCREMENT, publickey TEXT, date TIMESTAMP, name TEXT, state BLOB, c1 BLOB, c2 BLOB);
	CREATE TABLE IF NOT EXISTS verifications (id INTEGER PRIMARY KEY AUTOINCREMENT, publickey TEXT, who TEXT, date TIMESTAMP, how TEXT, name TEXT, signature TEXT);
	CREATE TABLE IF NOT EXISTS conversations (id INTEGER PRIMARY KEY AUTOINCREMENT, publickey TEXT, title TEXT, date_creation TIMESTAMP, date_last_message TIMESTAMP, c1 BLOB, c2 BLOB);
	CREATE TABLE IF NOT EXISTS messages (id INTEGER PRIMARY KEY AUTOINCREMENT, conversation_id INTEGER, date TIMESTAMP, sender TEXT, message BLOB);
	`
	_, err = ds.db.Exec(createStatement)
	if err != nil {
		panic(err)
	}

	// defer db.Close() // we never close the db
}

func (ds *databaseState) getMessages() {
	ds.queryMutex.Lock()
	defer ds.queryMutex.Unlock()

	selectStatement := "SELECT * FROM conversations;"
	_, err := ds.db.Exec(selectStatement)
	if err != nil {
		panic(err)
	}
}

func (ds *databaseState) getThreadRatchetStates(bobAddress string) ([]byte, []byte, error) {
	//
	ds.queryMutex.Lock()
	defer ds.queryMutex.Unlock()
	// query
	stmt, err := ds.db.Prepare("SELECT state, c1, c2 FROM contacts WHERE publickey = ?;")
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
func (ds *databaseState) getSessionKeys(convoId uint64, bobAddress string) ([]byte, []byte, error) {
	ds.queryMutex.Lock()
	defer ds.queryMutex.Unlock()

	stmt, err := ds.db.Prepare("SELECT state, c1, c2 FROM conversations WHERE id=? AND publickey=?")
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
func (ds *databaseState) updateSessionKeys(convoId uint64, bobAddress string, c1, c2 []byte) {
	ds.queryMutex.Lock()
	defer ds.queryMutex.Unlock()

	if c1 == nil && c2 == nil {
		panic("ssyk: at least one session key must be defined in order to call updateSessionKeys")
	}
	// c1 by default
	sessionkey := c1
	query := "UPDATE conversations SET c1=? WHERE id=? AND publickey=?"
	if c1 == nil {
		sessionkey = c2
		query := "UPDATE conversations SET c2=? WHERE id=? AND publickey=?"
	}
	stmt, err := ds.db.Prepare(query)
	if err != nil {
		panic(err)
	}
	_, err = stmt.Exec(sessionkey, convoId, bobAddress)
	if err != nil {
		panic(err)
	}
}

func (ds *databaseState) getLastConvoId() uint64 {
	ds.queryMutex.Lock()
	defer ds.queryMutex.Unlock()

	selectStatement := "SELECT id FROM conversations ORDER BY id DESC LIMIT 1;"
	rows, err := ds.db.Query(selectStatement)
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

func (ds *databaseState) createConvo(bobAddress, title string, sessionkey1, sessionkey2 []byte) uint64 {
	ds.queryMutex.Lock()
	defer ds.queryMutex.Unlock()

	// (id INTEGER PRIMARY KEY AUTOINCREMENT, publickey TEXT, title TEXT, date_creation TIMESTAMP, date_last_message TIMESTAMP, c1 BLOB, c2 BLOB);
	stmt, err := ds.db.Prepare("INSERT INTO conversations VALUES(NULL, ?, ?, ?, ?, ?, ?);")
	if err != nil {
		panic(err)
	}
	res, err := stmt.Exec(bobAddress, title, bobAddress, title, time.Now(), time.Now(), sessionkey1, sessionkey2)
	if err != nil {
		panic(err)
	}
	// return convoId created
	convoId, _ := res.LastInsertId()
	return convoId // TODO: returns a int64, does that mean that it's the maximum value that I can store in sqlite?
	// TODO: I can probably have a uint32 as value anyway, nobody is going to reach that
}

func (ds *databaseState) updateThreadRatchetStates(bobAddress string, ts1, ts2 []byte) {
	ds.queryMutex.Lock()
	defer ds.queryMutex.Unlock()

	if ts1 == nil && ts2 == nil {
		panic("ssyk: at least one session key must be defined in order to call updateSessionKeys")
	}
	// c1 by default
	threadState := ts1
	query := "UPDATE contacts SET c1=? WHERE publickey=?"
	if c1 == nil {
		sessionkey = c2
		query := "UPDATE contacts SET c2=? WHERE publickey=?"
	}
	stmt, err := ds.db.Prepare(query)
	if err != nil {
		panic(err)
	}
	_, err = stmt.Exec(threadState, bobAddress)
	if err != nil {
		panic(err)
	}
}

func (ds *databaseState) updateTitle(convoId uint64, bobAddress, title string) {

}
