package main

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"

	disco "github.com/mimoo/disco/libdisco"
)

// the big init function
func initSasayaki(passphrase string) (keyPair *disco.KeyPair, err error) {
	initSasayakiFolder()
	return initKeyPair(string(passphrase))
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
