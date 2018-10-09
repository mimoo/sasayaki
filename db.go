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
// - date_creation: metadata
// - date_last_message: metadata
// - publickey: of the account
// - sessionkey: state after the last message
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
	"path/filepath"
	"sync"
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
	CREATE TABLE IF NOT EXISTS conversations (id INTEGER PRIMARY KEY AUTOINCREMENT, date_creation TIMESTAMP, date_last_message TIMESTAMP, publickey TEXT, sessionkey BLOB);
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
