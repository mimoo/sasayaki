package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"golang.org/x/crypto/ssh/terminal"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// flags
	CLIenabled := flag.Bool("cli", false, "run Sasayaki in the terminal")
	addressUI := flag.String("port", "7474", "the address port of the web UI running on localhost (default 7474)")
	flag.Parse()

	if *CLIenabled {
		fmt.Println("Welcome to Sasayaki.")
		fmt.Println("In order to encrypt information at rest on your computer, please enter a passphrase:")
		passphrase, err := terminal.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			panic(err)
		}

		// init ~/.sasayaki folder and fetch config + keypair
		var config configuration
		config, ssyk.keyPair = initSasayaki(string(passphrase))
		ssyk.myAddress = ssyk.keyPair.ExportPublicKey()

		// init database
		initDatabaseManager()

		// if we don't have a hub address, we ask
		if config.HubAddress == "" {
			fmt.Println("What is the Hub address?")
			if _, err := fmt.Scanf("%s", config.HubAddress); err != nil {
				panic(err)
			}
		}

		// if we don't have a hub public key, we ask
		if config.HubPublicKey == "" {
			fmt.Println("What is the Hub public key?")
			var pubkeyHex string
			if n, err := fmt.Scanf("%s", config.HubPublicKey); err != nil || n != 64 {
				panic(err)
			}
		}

		hubPublicKey, err := hex.DecodeString(config.HubPublicKey)
		if err != nil {
			panic(err)
		}

		// init hub
		initHubManager(config.HubAddress, hubPublicKey)

		// Information
		fmt.Println("this is your public key:", ssyk.keyPair.ExportPublicKey())
		fmt.Println("this is the current config:", config)

		//
	} else {

		fmt.Println("not implemented yet")
		return

		// set address for the web UI
		addressUI := "127.0.0.1:" + *addressUI
		// serve the local webpage
		serveLocalWebPage(addressUI)

		// TODO: Create server at 127.0.0.1:nextOpenPort
		// TODO: serve a one-page js that removes the authToken and stores it in
		// TODO:use websockets for messages? (if I want to emulate email I can just use websocket as push notification)
		// TODO: package the app so that it's launched in the menu bar, not from a terminal

		//
		//
		// PUSH NOTIFICATIONS
		//

		/*
			// Dial the port 6666 of localhost
			notification, err := disco.Dial("tcp", "127.0.0.1:7475", &clientConfig)
			if err != nil {
				fmt.Println("client can't connect to server:", err)
				return
			}
			defer notification.Close()
			fmt.Println("connected to", notification.RemoteAddr())

			fmt.Println("connected to the Sasayaki Hub")
			defer notification.Close()

			// send our publickey
			notification.Write([]byte(keyPair.ExportPublicKey()))

			// receive push notifications
			var buffer [1]byte
			for {
				_, err := notification.Read(buffer[:])
				if err != io.EOF {
					fmt.Println("sasayaki: server closed the connection")
					break
				} else if err != nil {
					panic(err)
				}

				switch buffer[0] {
				case 0:
					fmt.Println("sasayaki: new contact request")
				case 1:
					fmt.Println("sasayaki: new conversation")
				case 2:
					fmt.Println("sasayaki: new message")
				default:
					fmt.Println("sasayaki: notification message not understood")
				}
			}
		*/
	}

	//
	fmt.Println("Bye bye!")
}

func openbrowser(url string) {
	switch runtime.GOOS {
	case "linux":
		exec.Command("xdg-open", url).Start()
	case "windows":
		exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		exec.Command("open", url).Start()
	}
}
