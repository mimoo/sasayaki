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
