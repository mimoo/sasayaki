package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"runtime"

	disco "github.com/mimoo/disco/libdisco"
)

// the big init function
func initSasayaki(passphrase string) (configuration, *disco.KeyPair) {
	initSasayakiFolder()
	initSasayakiDatabase()
	config := initConfiguration()
	keyPair, err := initKeyPair(string(passphrase))
	if err != nil {
		panic(err)
	}
	return config, keyPair
}

type configuration struct {
	addressUI string `json:"address_ui"`

	HubAddress   string `json:"hub_address"`
	HubPublicKey string `json:"hub_publickey"`
}

// read json file
func initConfiguration() configuration {
	home := sasayakiFolder()
	configFile := filepath.Join(home, "configuration.json")
	// this will create the file if it doesn't exist
	f, err := os.OpenFile(configFile, os.O_RDONLY|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}
	f.Close()
	// read the file
	configJSON, err := ioutil.ReadFile(configFile)
	if err != nil {
		panic(err)
	}
	// parse it
	cfg := configuration{}
	json.Unmarshal(configJSON, &cfg)

	// TODO: remove this default
	if len(cfg.HubPublicKey) == 0 {
		cfg.HubPublicKey = "1274e5b61840d54271e4144b80edc5af946a970ef1d84329368d1ec381ba2e21"
	}
	// TODO: remove this default
	if cfg.HubAddress == "" {
		cfg.HubAddress = "127.0.0.1:7474"
	}
	if cfg.addressUI == "" {
		cfg.addressUI = "127.0.0.1:7473"
	}

	//
	return cfg
}

// init ~/.sasayaki folder
func initSasayakiFolder() {
	home := sasayakiFolder()
	// create ~/.sasayaki if it doesn't exists
	if _, err := os.Stat(home); os.IsNotExist(err) {
		os.MkdirAll(home, 0770) // user | group | all
	}
	// create ~/.sasayaki/keys if it doesn't exists
	keyFolder := filepath.Join(home, "keys")
	if _, err := os.Stat(keyFolder); os.IsNotExist(err) {
		fmt.Println("sasayaki: creating configuration folder at", home)
		os.MkdirAll(keyFolder, 0770) // user | group | all
	}
}

// init database tables
func initSasayakiDatabase() {
	// Contacts
	// - id
	// - publickey: of the account
	// - date: metadata
	// - name: hector
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

	location := filepath.Join(sasayakiFolder(), "database.db")
	db, err := sql.Open("sqlite3", location)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	createStatement := `
	CREATE TABLE IF NOT EXISTS contacts (id INTEGER PRIMARY KEY AUTOINCREMENT, publickey TEXT, date TIMESTAMP, name TEXT);
	CREATE TABLE IF NOT EXISTS verifications (id INTEGER PRIMARY KEY AUTOINCREMENT, publickey TEXT, who TEXT, date TIMESTAMP, how TEXT, name TEXT, signature TEXT);
	CREATE TABLE IF NOT EXISTS conversations (id INTEGER PRIMARY KEY AUTOINCREMENT, date_creation TIMESTAMP, date_last_message TIMESTAMP, publickey TEXT, sessionkey TEXT);
	CREATE TABLE IF NOT EXISTS messages (id INTEGER PRIMARY KEY AUTOINCREMENT, conversation_id INTEGER, date TIMESTAMP, sender TEXT, message TEXT);
	`
	_, err = db.Exec(createStatement)
	if err != nil {
		panic(err)
	}
}

// init keypair
func initKeyPair(passphrase string) (*disco.KeyPair, error) {
	// location
	location := filepath.Join(sasayakiFolder(), "/keys/keypair")
	// create ~/.sasayaki/keys/keyPair
	if _, err := os.Stat(location); os.IsNotExist(err) {
		fmt.Println("sasayaki: generating a keypair for new user")
		return disco.GenerateAndSaveDiscoKeyPair(location, passphrase)
	} else { // if it already exists, load it
		keyPair, err := disco.LoadDiscoKeyPair(location, passphrase)
		if err != nil {
			return nil, errors.New("sasayaki: Cannot decrypt keyPair with given passphrase")
		}
		return keyPair, nil
	}
}

// get the ~ folder at runtime (os-dependent)
func sasayakiFolder() string {
	home := homeDir()
	if runtime.GOOS == "windows" {
		// what about using the previous home and doing this instead?
		// return filepath.Join(home, "AppData", "Roaming", "Sasayaki")
		home = filepath.Join(os.Getenv("HOMEDRIVE"), os.Getenv("HOMEPATH"))
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return filepath.Join(home, "sasayaki")
	}

	return filepath.Join(home, ".sasayaki")
}

// gets the home directory
func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}
