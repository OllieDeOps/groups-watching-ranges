package main

// Test the connection in terminal: echo -n "test out the server" | nc localhost 3333

import (
	"fmt"
	"net"
	"os"
)

const (
	connHost = "localhost"
	connPort = "3333"
	connType = "tcp"
)

func main() {
	// Listen for incoming connections.
	l, err := net.Listen(connType, connHost+":"+connPort)
	if err != nil {
		fmt.Println("Error listening: ", err.Error())
		os.Exit(1)
	}

	// Close the listener when the app closes.
	defer l.Close()
	fmt.Println("Listening on " + connHost + ":" + connPort)
	for {
		// LIsten for an incoming connection.
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}
		// Handle connections in a new goroutine.
		go handleRequest(conn)
	}
}

// Handles incoming requests
func handleRequest(conn net.Conn) {
	// Make a buffer to hold incoming data.
	buf := make([]byte, 1024)

	//Read the incoming connection into the buffer.
	_, err := conn.Read(buf)
	if err != nil {
		fmt.Println("Error reading: ", err.Error())
	}

	// Send a response back to person contacting us.
	conn.Write([]byte("Message received.\n"))

	// Close the connection when finished with it.
	conn.Close()
}
