package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/golang/protobuf/proto"
	s "github.com/mimoo/sasayaki/serialization"

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

	// TODO: let the user enter the passphrase from the webapp

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
	fmt.Println("To use Sasayaki, open the following url in your favorite browser:", url)
	openbrowser(url)

	// TODO: we should still be able to access the webapp if this doesn't work v
	// or better, this GET request should get triggered by the webapp, and if it doesn't work tell the user
	// that perhaps the server is not configured correctly

	// connect to the Sasayaki hub server to request last messages
	remoteKey, _ := hex.DecodeString("1274e5b61840d54271e4144b80edc5af946a970ef1d84329368d1ec381ba2e21")
	clientConfig := disco.Config{
		KeyPair:              keyPair,
		HandshakePattern:     disco.Noise_IK,
		RemoteKey:            remoteKey, // TODO: should be in a config file
		StaticPublicKeyProof: []byte{},
	}

	// TODO: have a config file for this ip
	// TODO: perhaps even a configurator on first launch that asks for it
	conn, err := disco.Dial("tcp", "127.0.0.1:7474", &clientConfig)
	if err != nil {
		fmt.Println("can't connect to hub:", err)
		return
	}

	req := &s.Request{RequestType: s.Request_GetPendingMessages}

	data, err := proto.Marshal(req)
	fmt.Println("going to send", data)
	if err != nil {
		log.Panic("marshaling error: ", err)
	}
	_, err = conn.Write(data)

	if err != nil {
		panic("can't write to the server") // TODO: should I really panic here?
	}

	var buffer [3000]byte
	n, err := conn.Read(buffer[:])
	if err != nil {
		panic("can't read from the server") // TODO: should I really panic here?
	}
	fmt.Println("response received:", buffer[:n])

	conn.Close()

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
	fmt.Fprintf(w, "<html><body>Hello <code>%s</code><br><a href=#>add someone</a></body></html>", keyPair.ExportPublicKey())
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
