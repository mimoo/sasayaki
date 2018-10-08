package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"sync"

	"golang.org/x/crypto/ssh/terminal"

	_ "github.com/mattn/go-sqlite3"
	disco "github.com/mimoo/disco/libdisco"
)

type sasayakiState struct {
	conn net.Conn

	queryMutex sync.Mutex // one hub query at a time

	config  configuration
	keyPair *disco.KeyPair
}

var state sasayakiState

func main() {
	// Welcome + Passphrase
	// TODO: ask the passphrase in the browser perhaps?
	fmt.Println("Welcome to Sasayaki.")
	fmt.Println("In order to encrypt information at rest on your computer, please enter a passphrase:")
	passphrase, err := terminal.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}

	// Initialization
	state.config, state.keyPair = initSasayaki(string(passphrase))

	//
	fmt.Println("this is your public key:", state.keyPair.ExportPublicKey())
	fmt.Println("this is the current config:", state.config)

	// TODO: Create server at 127.0.0.1:nextOpenPort
	// TODO: serve a one-page js that removes the authToken and stores it in
	// TODO:use websockets for messages? (if I want to emulate email I can just use websocket as push notification)
	// TODO: package the app so that it's launched in the menu bar, not from a terminal
	var token [16]byte
	_, err = rand.Read(token[:])
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/", handler)
	url := fmt.Sprintf("http://localhost:7373/?token=%s", base64.StdEncoding.EncodeToString(token[:]))
	go func() {
		panic(http.ListenAndServe("localhost:7373", nil))
	}()

	// open on browser
	fmt.Println("To use Sasayaki, open the following url in your favorite browser:", url)
	openbrowser(url)

	//
	// /get_new_messages (sorted)
	//
	// -> [
	// 	{convo_id: "random_guid", message: "E(date|content, AD=to+from)", from: "publickey"},
	// 	{...},
	// 	{...},
	// ]
	//
	// /start_conversation
	//
	// (we trust the date received)
	//
	// POST convo_id: "random_guid", to: "publickey", message: "E(date|content, AD=to+from)"

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
	//
	fmt.Println("Bye bye!")
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<html><body>Hello <code>%s</code><br><a href=#>add someone</a></body></html>", state.keyPair.ExportPublicKey())
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
