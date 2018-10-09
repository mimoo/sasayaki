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

package main

import (
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/gorilla/mux"
)

const (
	mediaPath       = "web"
	messageMaxChars = 10000
)

func serveLocalWebPage(localAddress string) {
	r := mux.NewRouter()
	r.HandleFunc("/", getApp).Methods("GET")
	r.HandleFunc("/get_new_messages", getNewMessages).Methods("GET")
	r.HandleFunc("/send_message", sendMessage).Methods("POST")

	panic(http.ListenAndServe(localAddress, r))
}

// verifyToken obtains the Sasayaki-Token header from the request
// then constant-time compares it to what it should be (a random 16-byte token
// that we generated when we started the application)
func verifyToken(r *http.Request) bool {
	givenToken := r.Header.Get("Sasayaki-Token")
	decodedToken, err := base64.StdEncoding.DecodeString(givenToken)
	if err != nil {
		return false
	}
	if subtle.ConstantTimeCompare(ss.token[:], decodedToken) == 1 {
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

func getApp(w http.ResponseWriter, r *http.Request) {
	indexPageLocation := filepath.Join(mediaPath, "index.html")

	tmpl := template.Must(template.ParseFiles(indexPageLocation))
	tmpl.Execute(w, indexData{
		Identity: ss.keyPair.ExportPublicKey(),
	})

}

//
// JSON API
//

// http get http://127.0.0.1:7473/get_new_messages Sasayaki-Token:wZ8VHXeKBoSrQ+m5sGnCFQ==
func getNewMessages(w http.ResponseWriter, r *http.Request) {
	// verify auth token
	if !verifyToken(r) {
		fmt.Fprintf(w, "You need to enter the correct auth token")
		return
	}
	// return sample
	json.NewEncoder(w).Encode(map[string]string{
		"message": "one message",
	})
}

// http post http://127.0.0.1:7473/send_message Sasayaki-Token:wZ8VHXeKBoSrQ+m5sGnCFQ== id=1 convo_id=5 to=pubkey

type sendMessageReq struct {
	Id        string `json:"id"`       // 64-bit?
	ConvoId   string `json:"convo_id"` // 64-bit?
	ToAddress string `json:"to_address"`
	Content   string `json:"content"`
}

func sendMessage(w http.ResponseWriter, r *http.Request) {
	// verify auth token
	if !verifyToken(r) {
		json.NewEncoder(w).Encode(map[string]string{"error": "You need to enter the correct auth token"})
		return
	}
	// parse request
	decoder := json.NewDecoder(r.Body)
	var req sendMessageReq
	err := decoder.Decode(&req)
	if err != nil || req.Id == "" || req.ConvoId == "" || len(req.ToAddress) != 64 || req.Content == "" || len(req.Content) > messageMaxChars {
		log.Println("couldn't decode sendMessage req:", err)
		json.NewEncoder(w).Encode(map[string]string{"error": "Couldn't parse the request"})
		return
	}
	// to Number
	reqId, err := strconv.ParseUint(req.Id, 10, 64)
	if err != nil {
		log.Println("couldn't decode sendMessage req id:", err)
		json.NewEncoder(w).Encode(map[string]string{"error": "Couldn't parse the request id"})
		return
	}
	convoId, err := strconv.ParseUint(req.ConvoId, 10, 64)
	if err != nil {
		log.Println("couldn't decode sendMessage req convo id:", err)
		json.NewEncoder(w).Encode(map[string]string{"error": "Couldn't parse the request convo id"})
		return
	}
	// TODO: encrypt
	// TODO: 1. fetch shared secret for the convo
	// TODO: 2. is this a new convo? if so, then derive a new shared secret with c1 or c2
	// TODO: 3. encrypt the content with that s
	content := []byte(req.Content)
	// use the proxy
	success, err := hs.sendMessage(reqId, convoId, req.ToAddress, content)
	// return
	json.NewEncoder(w).Encode(map[string]string{
		"success": strconv.FormatBool(success),
		"error":   err.Error(),
	})
}

/*
r.HandleFunc("/books/{title}/page/{page}", serveOther)
func serveOther(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	title := vars["title"]
	page := vars["page"]

	fmt.Fprintf(w, "You've requested the book: %s on page %s\n", title, page)
}
*/
