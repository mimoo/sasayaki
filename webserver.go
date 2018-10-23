//
// Local Web Server
// ================
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
	"encoding/hex"
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

//
// JSON APIs
//

// send_message
type sendMessageReq struct {
	ConvoId   string `json:"convo_id"`
	ToAddress string `json:"to_address"`
	Content   string `json:"content"`
}

// set_passphrase
type passphraseRequest struct {
	Passphrase string `json:"passphrase"`
}

// add_contact
type addContactReq struct {
	ToAddress string `json:"to_address"`
	Name      string `json:"name"`
}

// accept_contact
type ackContactReq struct {
	FromAddress           string `json:"from_address"`
	Name                  string `json:"name"`
	FirstHandshakeMessage string `json:"first_handshake_message"`
}

// serveLocalWebPage is the main function serving the single-page javascript webapp
// and the different JSON APIs
func serveLocalWebPage(localAddress string) {
	r := mux.NewRouter()
	// main page
	r.HandleFunc("/", web.getApp).Methods("GET")
	// configuration
	r.HandleFunc("/set_passphrase", web.setPassphrase).Methods("POST")
	r.HandleFunc("/set_configuration", web.setConfiguration).Methods("POST")
	r.HandleFunc("/get_configuration", web.getConfiguration).Methods("GET")
	// contacts
	r.HandleFunc("/add_contact", web.addContact).Methods("POST")
	r.HandleFunc("/accept_contact_request", web.acceptContactRequest).Methods("POST")
	// messages
	r.HandleFunc("/get_new_message", web.getNewMessage).Methods("GET")
	r.HandleFunc("/send_message", web.sendMessage).Methods("POST")

	// token
	if _, err := rand.Read(web.token[:]); err != nil {
		panic(err)
	}

	url := fmt.Sprintf("http://%s/?token=%s", localAddress, base64.URLEncoding.EncodeToString(web.token[:]))
	// open on browser
	fmt.Println("To use Sasayaki, open the following url in your favorite browser:", url)

	if !ssyk.debug {
		openbrowser(url) // TODO: should we really open this before starting the server?
	}

	// listen and serve
	panic(http.ListenAndServe(localAddress, r))
}

// verifyToken obtains the Sasayaki-Token header from the request
// then constant-time compares it to what it should be (a random 16-byte token
// that we generated when we started the application)
func verifyToken(givenToken string) bool {
	if ssyk.debug {
		return true
	}
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
	//		Identity: ssyk.keyPair.ExportPublicKey(), // TODO: can't display that as we haven't initliazed
	})

}

//
// JSON API
//

// getNewMessage returns one message at a time, you need to call it several time in order to retrieve
// all your messages. It's not ideal but heh, it works for now.
// http post http://127.0.0.1:7473/send_message Sasayaki-Token:dwl0R9o2SwuZQIAWHv-== id=5 convo_id=6 to_address="12052512a0e1cf14092224dba5a88c98ad8c5efe23f7794a122b9f0268499a10"  content="hey"
func (web webState) getNewMessage(w http.ResponseWriter, r *http.Request) {
	// initialized?
	if !ssyk.initialized {
		json.NewEncoder(w).Encode(map[string]string{"error": "Sasayaki needs to be initialized first"})
		return
	}
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
// sendMessage can be used with an empty convo_id in order to create a new thread
func (web webState) sendMessage(w http.ResponseWriter, r *http.Request) {
	// initialized?
	if !ssyk.initialized {
		json.NewEncoder(w).Encode(map[string]string{"error": "Sasayaki needs to be initialized first"})
		return
	}
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

	// send message via sasayaki core algorithm
	if convoId, err := ssyk.sendMessage(msg); err != nil {
		json.NewEncoder(w).Encode(map[string]string{
			"success": "false",
			"error":   err.Error(),
		})
	} else {
		json.NewEncoder(w).Encode(map[string]string{
			"success":  "true",
			"convo_id": convoId,
		})
	}
}

// http post http://127.0.0.1:7473/set_passphrase Sasayaki-Token:dwl0R9o2SwuZQIAWHv-== id=5 convo_id=6 to_address="12052512a0e1cf14092224dba5a88c98ad8c5efe23f7794a122b9f0268499a10"  passphrase="prout"
func (web webState) setPassphrase(w http.ResponseWriter, r *http.Request) {
	// already initialized?
	if ssyk.initialized {
		json.NewEncoder(w).Encode(map[string]string{"error": "Sasayaki is already initialized"})
		return
	}
	// parse request
	decoder := json.NewDecoder(r.Body)
	var passphraseReq passphraseRequest
	err := decoder.Decode(&passphraseReq)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "Couldn't parse the request"})
		return
	}

	// init
	var config *configuration
	config, ssyk.keyPair, err = initSasayaki(string(passphraseReq.Passphrase))
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "Passphrase is incorrect"})
		return
	}
	ssyk.myAddress = ssyk.keyPair.ExportPublicKey()

	// init database
	initDatabaseManager()

	// init hub
	hubPublicKey, err := hex.DecodeString(config.HubPublicKey)
	if err != nil || len(hubPublicKey) != 32 {
		json.NewEncoder(w).Encode(map[string]string{"error": "Couldn't parse the request"})
		return
	}

	initHubManager(config.HubAddress, hubPublicKey)

	// done
	ssyk.initialized = true

	//
	json.NewEncoder(w).Encode(map[string]string{"success": "true"})
}

// http get http://127.0.0.1:7473/get_configuration Sasayaki-Token:dwl0R9o2SwuZQIAWHv-==
func (web webState) getConfiguration(w http.ResponseWriter, r *http.Request) {
	// initialized?
	if !ssyk.initialized {
		json.NewEncoder(w).Encode(map[string]string{"error": "Sasayaki needs to be initialized first"})
		return
	}

	//
	json.NewEncoder(w).Encode(map[string]string{
		"myAddress":     ssyk.myAddress,
		"hub_address":   hub.hubAddress,
		"hub_publickey": hex.EncodeToString(hub.hubPublicKey),
	})
}

// http post http://127.0.0.1:7473/set_configuration Sasayaki-Token:dwl0R9o2SwuZQIAWHv-== id=5 convo_id=6 to_address="12052512a0e1cf14092224dba5a88c98ad8c5efe23f7794a122b9f0268499a10"  hub_address="127.0.0.1:7474" hub_publickey="1274e5b61840d54271e4144b80edc5af946a970ef1d84329368d1ec381ba2e21"
func (web webState) setConfiguration(w http.ResponseWriter, r *http.Request) {
	// initialized?
	if !ssyk.initialized {
		json.NewEncoder(w).Encode(map[string]string{"error": "Sasayaki needs to be initialized first"})
		return
	}
	// parse request
	decoder := json.NewDecoder(r.Body)
	var cfgReq configuration
	err := decoder.Decode(&cfgReq)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "Couldn't parse the request"})
		return
	}

	// init hub
	hubPublicKey, err := hex.DecodeString(cfgReq.HubPublicKey)
	if err != nil || len(hubPublicKey) != 32 {
		json.NewEncoder(w).Encode(map[string]string{"error": "hub public key is incorrect"})
		return
	}

	initHubManager(cfgReq.HubAddress, hubPublicKey)

	// save configuration
	cfgReq.updateConfiguration()

	//
	json.NewEncoder(w).Encode(map[string]string{"success": "true"})
}

func (web webState) addContact(w http.ResponseWriter, r *http.Request) {
	// initialized?
	if !ssyk.initialized {
		json.NewEncoder(w).Encode(map[string]string{"error": "Sasayaki needs to be initialized first"})
		return
	}
	// parse request
	decoder := json.NewDecoder(r.Body)
	var addReq addContactReq
	err := decoder.Decode(&addReq)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "Couldn't parse the request"})
		return
	}

	// pass the request to core
	if err := ssyk.addContact(addReq.ToAddress, addReq.Name); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	//
	json.NewEncoder(w).Encode(map[string]string{"success": "true"})
}

func (web webState) acceptContactRequest(w http.ResponseWriter, r *http.Request) {
	// initialized?
	if !ssyk.initialized {
		json.NewEncoder(w).Encode(map[string]string{"error": "Sasayaki needs to be initialized first"})
		return
	}
	// parse request
	decoder := json.NewDecoder(r.Body)
	var ackReq ackContactReq
	err := decoder.Decode(&ackReq)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "Couldn't parse the request"})
		return
	}
	// decode first handshake message
	handshakeMsg, err := hex.DecodeString(ackReq.FirstHandshakeMessage)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "Couldn't parse the handshake message"})
		return
	}

	// pass the request to core
	if err := ssyk.acceptContact(ackReq.FromAddress, ackReq.Name, handshakeMsg); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	//
	json.NewEncoder(w).Encode(map[string]string{"success": "true"})
}
