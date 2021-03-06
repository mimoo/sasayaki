package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"golang.org/x/crypto/ssh/terminal"

	_ "github.com/mattn/go-sqlite3"
)

var debug bool

func main() {
	// flags
	CLIenabled := flag.Bool("cli", false, "run Sasayaki in the terminal")
	// TODO: change to port 0?
	addressUI := flag.String("port", "7473", "the address port of the web UI running on localhost (default 7474)")
	debug := flag.Bool("debug", false, "debug")
	flag.Parse()
	debug = *debug

	if *CLIenabled {
		fmt.Println("Welcome to Sasayaki.")
		fmt.Println("In order to encrypt information at rest on your computer, please enter a passphrase:")
		passphrase, err := terminal.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			fmt.Println(err)
			return
		}

		// TODO: ideally, we would use the Hub as an OPRF here, so that our passphrase is not too weak
		// (see PASS, OPAQUE, SPHINX, MAKWA, etc.)
		// + rate-limit on the server-side

		// init ~/.sasayaki folder and fetch config + keypair
		var config *configuration
		config, ssyk.keyPair, err = initSasayaki(string(passphrase))
		if err != nil {
			fmt.Println(err)
			return
		}

		// if we don't have a hub address, we ask
		var updateCfg bool
		if config.HubAddress == "" {
			fmt.Println("What is the Hub address?")
			if _, err := fmt.Scanf("%s", &(config.HubAddress)); err != nil {
				fmt.Println(err)
				return
			}
			updateCfg = true
		}

		// if we don't have a hub public key, we ask
		if config.HubPublicKey == "" {
			fmt.Println("What is the Hub public key?")
			if _, err := fmt.Scanf("%s", &(config.HubPublicKey)); err != nil {
				fmt.Println(err)
				return
			}
			updateCfg = true
		}

		// save configuration if there are changes
		if updateCfg {
			config.updateConfiguration()
		}

		// Information
		fmt.Println("this is your public key:", ssyk.keyPair.ExportPublicKey())
		fmt.Println("this is the current config:", config)

		// init sasayakiState
		initSasayakiState(keyPair, config)

	} else {

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
