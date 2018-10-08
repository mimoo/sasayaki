package main

import (
	"flag"
	"fmt"
	"log"

	disco "github.com/mimoo/disco/libdisco"
)

const (
	defaultKeyPairFile = "server.keypair"
)

func main() {
	// Flags
	genKeyPair := flag.Bool("gen_keypair", false, "generate a keypair for the server")
	keyPairFile := flag.String("keypair_file", defaultKeyPairFile, "sets the server.keypair location (default to current directory)")
	runServer := flag.Bool("run", false, "runs the Sasayaki Server")

	flag.Parse()

	// Init
	fmt.Println("==== Sasayaki Server ====")

	if *genKeyPair {
		_, err := disco.GenerateAndSaveDiscoKeyPair(*keyPairFile, "")
		if err != nil {
			panic("server cannot store keypair")
		}
		fmt.Println("Sasayaki server successfuly generated private key at location ", *keyPairFile)
		return
	}

	if !*runServer {
		flag.PrintDefaults()
		return
	}

	keyPair, err := disco.LoadDiscoKeyPair(*keyPairFile, "")
	if err != nil {
		panic("server cannot load keypair")
		return
	}
	fmt.Println("Sasayaki Hub's public key:", keyPair.ExportPublicKey())

	//
	// the RPC API
	//
	// TODO: timeouts
	serverConfig := disco.Config{
		HandshakePattern:  disco.Noise_IK,
		KeyPair:           keyPair,
		PublicKeyVerifier: func(publicKey, proof []byte) bool { return true },
	}

	// listen on port 6666
	listener, err := disco.Listen("tcp", "127.0.0.1:7474", &serverConfig)
	if err != nil {
		fmt.Println("RPC server cannot setup a listener:", err)
		return
	}
	addr := listener.Addr().String()
	fmt.Println("RPC server listening on:", addr)

	// currently only accept one client
	go sasayakiServer(listener)

	//
	// Push notifications
	//

	// listen on port 6666
	// TODO: different timeouts? keep-alive?
	// I think to avoid DoS I need keep-alive
	notification, err := disco.Listen("tcp", "127.0.0.1:7475", &serverConfig)
	if err != nil {
		fmt.Println("notification server cannot setup a listener:", err)
		return
	}
	addr = notification.Addr().String()
	fmt.Println("notification server listening on:", addr)

	// currently only accept one client
	for {
		conn, err := notification.Accept()
		if err != nil {
			log.Println("notification server couldn't accept client:", err)
			continue
		}
		go handleNotificationClient(conn)
	}
}
