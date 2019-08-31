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

	// connect to this socket
	conn, _ := net.Dial(connType, connHost+":"+connPort)
	for {
		// read in input from stdin
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("-> ")
		text, _ := reader.ReadString('\n')
		// send to socket
		fmt.Fprintf(conn, text)
		if text == "STOP\n" {
			os.Exit(1)
		}
		// listen for reply
		message, _ := bufio.NewReader(conn).ReadString('\n')
		fmt.Print(message)
	}
}
