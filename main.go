package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"golang.org/x/crypto/ssh/terminal"

	_ "github.com/mattn/go-sqlite3"
	disco "github.com/mimoo/disco/libdisco"
)

const (
	serverIp   = "127.0.0.1"
	serverPort = "6861"
)

var (
	keyPair *disco.KeyPair
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
	keyPair, err = initSasayaki(string(passphrase))
	if err != nil {
		panic(err)
	}

	fmt.Println(keyPair)

	//
	// INIT CONFIGURATION
	// should be a map[key]value
	// that can be saved as a key-value store converted to a json file
	// don't think this file needs to be encrypted?
	//

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
		log.Printf("%q: %s\n", err, createStatement)
		return
	}

	// Create server at 127.0.0.1:nextOpenPort
	// -> open that url with default browser
	// -> with ?authToken={randomValue}
	// -> serve a one-page js that removes the authToken and stores it in
	// -> display the full url+token in the terminal?
	//    -> bad idea since it will be daemon later?
	// -> use websockets for messages? (if I want to emulate email I can just use websocket as push notification)
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
	fmt.Println("interface listening on", url)
	openbrowser(url)

	//
	//
	// connect to the Sasayaki "hub?" server
	// I think this should be done using TLS + gRPC instead
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

	// configure the Disco connection with Noise_IK
	clientConfig := disco.Config{
		KeyPair:              keyPair,
		HandshakePattern:     disco.Noise_IK,
		StaticPublicKeyProof: []byte{},
	}

	// Dial the port 6666 of localhost
	conn, err := disco.Dial("tcp", "127.0.0.1:6666", &clientConfig)
	if err != nil {
		fmt.Println("client can't connect to server:", err)
		return
	}
	defer conn.Close()
	fmt.Println("connected to", conn.RemoteAddr())

	fmt.Println("connected to the Sasayaki Hub")
	defer conn.Close()

	// send our publickey
	conn.Write([]byte(keyPair.ExportPublicKey()))

	// receive push notifications
	var buffer [1]byte
	for {
		_, err := conn.Read(buffer[:])
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

	//
	fmt.Println("Bye bye!")
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello %s!<br><a href=#>add someone</a>", keyPair.ExportPublicKey())
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
