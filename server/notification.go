package main

import "net"

func handleNotificationClient(conn net.Conn) {
	// save the client in a list of client somewhere

	//
	conn.Close()
}
