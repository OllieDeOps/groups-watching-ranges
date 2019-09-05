package main

import (
	"bufio"
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
	// Connect to this socket
	conn, _ := net.Dial(connType, connHost+":"+connPort)
	for {
		// Read in input from stdin
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("-> ")
		text, _ := reader.ReadString('\n')
		// Send to socket
		fmt.Fprintf(conn, text)
		// Listen for reply
		message, _ := bufio.NewReader(conn).ReadString('\n')
		fmt.Print(message)
		if message == "Disconnecting\n" {
			os.Exit(1)
		}
	}
}
