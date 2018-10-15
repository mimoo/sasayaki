//
// Web Server
// ==========
//
// This file takes care of:
//
// * serving the single-page web application to localhost
// * serving the JSON API that the web app can query
// * using the Encryption Manager (encryption.go) to encrypt and decrypt messages
// * using the Hub Manager (proxy.go) to forward requests to the Hub
// * using the Database Manager (db.go) to retrieve/store information
//
// This file can thus be seen as the core app, communicating between the thin webapp client and the hub
//
// ```
// web browser <--JSON--> web server <--PROTOBUF--> Hub
// ```
//
// Note that since the single-page web application is in javascript, uint64 numbers are not supported.
// Fortunately, we do not need to do arithmetic from the client, and we can pretend that these are strings.
// (We only need them as identifiers for conversations or messages.)
//

package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"
)

const (
	mediaPath       = "web"
	messageMaxChars = 10000
)

type webState struct {
	conn net.Conn

	token [16]byte // for the webapp
}

var web webState

// serveLocalWebPage is the main function serving the single-page javascript webapp
// and the different JSON APIs
func serveLocalWebPage(localAddress string) {
	// handlers
	r := mux.NewRouter()
	r.HandleFunc("/", web.getApp).Methods("GET")
	r.HandleFunc("/get_new_message", web.getNewMessage).Methods("GET")
	r.HandleFunc("/send_message", web.sendMessage).Methods("POST")

	// token
	if _, err := rand.Read(web.token[:]); err != nil {
		panic(err)
	}

	url := fmt.Sprintf("http://%s/?token=%s", localAddress, base64.URLEncoding.EncodeToString(web.token[:]))
	// open on browser
	fmt.Println("To use Sasayaki, open the following url in your favorite browser:", url)
	openbrowser(url) // TODO: should we really open this before starting the server?

	// listen and serve
	panic(http.ListenAndServe(localAddress, r))
}

// verifyToken obtains the Sasayaki-Token header from the request
// then constant-time compares it to what it should be (a random 16-byte token
// that we generated when we started the application)
func verifyToken(givenToken string) bool {
	// TODO: remove this return true :)
	return true
	decodedToken, err := base64.URLEncoding.DecodeString(givenToken)
	if err != nil {
		return false
	}
	if subtle.ConstantTimeCompare(web.token[:], decodedToken) == 1 {
		return true
	}
	return false
}

//
// Home page
//

type indexData struct {
	Identity string
}

func (web webState) getApp(w http.ResponseWriter, r *http.Request) {
	// get the GET request and the "token" parameter
	token := r.URL.Query().Get("token")
	// verify auth token
	if !verifyToken(token) {
		fmt.Fprintf(w, "You need to enter the correct auth token")
		return
	}
	// get html page
	indexPageLocation := filepath.Join(mediaPath, "index.html")
	// render the template
	tmpl := template.Must(template.ParseFiles(indexPageLocation))
	tmpl.Execute(w, indexData{
		Identity: ssyk.keyPair.ExportPublicKey(),
	})

}

//
// JSON API
//

// getNewMessage returns one message at a time, you need to call it several time in order to retrieve
// all your messages. It's not ideal but heh, it works for now.
// http post http://127.0.0.1:7473/send_message Sasayaki-Token:dwl0R9o2SwuZQIAWHv-== id=5 convo_id=6 to_address="12052512a0e1cf14092224dba5a88c98ad8c5efe23f7794a122b9f0268499a10"  content="hey"
func (web webState) getNewMessage(w http.ResponseWriter, r *http.Request) {
	// verify auth token
	if !verifyToken(r.Header.Get("Sasayaki-Token")) {
		fmt.Fprintf(w, "You need to enter the correct auth token")
		return
	}

	msg, err := ssyk.getNextMessage()

	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(msg)
}

// http post http://127.0.0.1:7473/send_message Sasayaki-Token:wZ8VHXeKBoSrQ+m5sGnCFQ== id=1 convo_id=5 to=pubkey

type sendMessageReq struct {
	Id        string `json:"id"` // 64-bit?
	ConvoId   string `json:"convo_id"`
	ToAddress string `json:"to_address"`
	Content   string `json:"content"`
}

func (web webState) sendMessage(w http.ResponseWriter, r *http.Request) {
	// verify auth token
	if !verifyToken(r.Header.Get("Sasayaki-Token")) {
		json.NewEncoder(w).Encode(map[string]string{"error": "You need to enter the correct auth token"})
		return
	}
	// parse request
	decoder := json.NewDecoder(r.Body)
	var req sendMessageReq
	err := decoder.Decode(&req)
	if err != nil || len(req.ToAddress) != 64 || req.Content == "" || len(req.Content) > messageMaxChars {
		log.Println("couldn't decode sendMessage req:", err)
		json.NewEncoder(w).Encode(map[string]string{"error": "Couldn't parse the request"})
		return
	}

	// use the proxy to forward the request to the hub
	msg := &plaintextMsg{
		ConvoId:     req.ConvoId,
		FromAddress: ssyk.myAddress,
		ToAddress:   req.ToAddress,
		Content:     req.Content,
	}
	if err := ssyk.sendMessage(msg); err != nil {
		json.NewEncoder(w).Encode(map[string]string{
			"success": "false",
			"error":   err.Error(),
		})
	} else {
		json.NewEncoder(w).Encode(map[string]string{
			"success": "true",
		})
	}
}
