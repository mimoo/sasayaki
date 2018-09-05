package main

import (
	"crypto/tls"
	"crypto/x509"
	"log"
	"net"
	"net/http"
)

func main() {
	//
	// the RPC API
	//
	http.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte("Hello, world\n"))
	})

	server := &http.Server{
		Addr: "localhost:7474",
		TLSConfig: &tls.Config{
			// Avoids most of the memorably-named TLS attacks
			MinVersion: tls.VersionTLS12,
			// Causes servers to use Go's default ciphersuite preferences,
			// which are tuned to avoid attacks. Does nothing on clients.
			PreferServerCipherSuites: true,
			// Only use curves which have constant-time implementations
			CurvePreferences: []tls.CurveID{
				tls.CurveP256,
			},
		},
	}

	err := server.ListenAndServeTLS("cert.pem", "key.pem")
	if err != nil {
		log.Fatal(err)
	}

	//
	// Push notifications
	//

	cert, err := tls.LoadX509KeyPair("cert.pem", "key.key")
	if err != nil {
		panic("server: can't load keys")
	}

	listener, err := tls.Listen("tcp", "localhost:7475", &tls.Config{
		Certificates:             []tls.Certificate{cert},
		PreferServerCipherSuites: true,
		MinVersion:               tls.VersionTLS12,
		CurvePreferences: []tls.CurveID{
			tls.CurveP256,
		},
	})
	if err != nil {
		panic(err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("server: accept: %s", err)
			break
		}
		log.Printf("server: accepted from %s", conn.RemoteAddr())
		tlscon, ok := conn.(*tls.Conn)
		if ok {
			log.Print("ok=true")
			state := tlscon.ConnectionState()
			for _, v := range state.PeerCertificates {
				log.Print(x509.MarshalPKIXPublicKey(v.PublicKey))
			}
		}
		go handleClient(conn)
	}
}

func handleClient(conn net.Conn) {
	// save the client in a list of client somewhere

	//
	conn.Close()
}
