package main

import (
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
	config := initConfiguration()
	keyPair, err := initKeyPair(string(passphrase))
	if err != nil {
		panic(err)
	}
	return config, keyPair
}

type configuration struct {
	HubAddress   string `json:"hub_address"`
	HubPublicKey string `json:"hub_publickey"`
}

// read json file
// TODO: should I encrypt stuff in there?
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

	if ssyk.debug {
		if len(cfg.HubPublicKey) == 0 {
			cfg.HubPublicKey = "1274e5b61840d54271e4144b80edc5af946a970ef1d84329368d1ec381ba2e21"
		}
		if cfg.HubAddress == "" {
			cfg.HubAddress = "127.0.0.1:7474"
		}
	}

	//
	return cfg
}

// write json file
func (cfg configuration) updateConfiguration() {
	home := sasayakiFolder()
	configFile := filepath.Join(home, "configuration.json")
	// this will create the file if it doesn't exist
	f, err := os.OpenFile(configFile, os.O_WRONLY, 0600)
	if err != nil {
		panic(err)
	}

	if err := json.NewEncoder(f).Encode(cfg); err != nil {
		panic(err)
	}

	f.Close()
}

// reset json file
func (cfg configuration) resetConfiguration() {
	home := sasayakiFolder()
	configFile := filepath.Join(home, "configuration.json")
	os.Remove(configFile)
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
