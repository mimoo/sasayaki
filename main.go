package main

import (
	"fmt"
	"os"

	"golang.org/x/crypto/ssh/terminal"
)

const (
	serverIp   = "127.0.0.1"
	serverPort = "6861"
)

func main() {
	// Welcome + Passphrase
	fmt.Println("Welcome to Sasayaki.")
	fmt.Println("In order to encrypt information at rest on your computer, please enter a passphrase:")
	passphrase, err := terminal.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	// Initialization
	keyPair, err := initSasayaki(string(passphrase))
	if err != nil {
		panic(err)
	}

	fmt.Println(keyPair)

	// Load contacts ~/.sasayaki/contacts/
	// as a json file?
	// {
	// 	{
	// 		"name": "david",
	// 		"pubkey": "...",
	// 		"verified": [...]
	// 	}
	// }
	// or better, sqlite3 file! but every row needs to be encrypted with our key

	// Load old messages ~/.sasayaki/messages/

	// Create server at 127.0.0.1:nextOpenPort
	// -> open that url with default browser
	// -> with ?authToken={randomValue}
	// -> serve a one-page js that removes the authToken and stores it in
	// -> display the full url+token in the terminal?
	//    -> bad idea since it will be daemon later?

	// Connect to the server and check new messages

}
