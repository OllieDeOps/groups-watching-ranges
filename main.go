package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
)

const (
	connHost = "localhost"
	connPort = "3333"
	connType = "tcp"
)

// Group defines the watcher
type Group struct {
	name     string
	watching []int32
}

var groups []Group

// ParsedCmd holds type-converted data sent over network to pass around
type ParsedCmd struct {
	rangeStart int32
	rangeEnd   int32
	groupName  string
}

func main() {
	// Listen for incoming connections.
	l, err := net.Listen(connType, connHost+":"+connPort)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	// Close the listener when app closes.
	defer l.Close()
	fmt.Println("Listening on " + connHost + ":" + connPort)

	for {
		// Listen for incoming connection.
		c, err := l.Accept()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		// Handle each connection in a new goroutine.
		go handleConnection(c)
	}
}

func handleConnection(c net.Conn) {
	fmt.Printf("Serving %s\n", c.RemoteAddr().String())
	for {
		// Our recieved string.
		netData, err := bufio.NewReader(c).ReadString('\n')
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		// Parse cmd ints
		cmd := strings.Fields(string(netData))
		rangeStart64, _ := strconv.ParseInt(cmd[1], 10, 64)
		rangeStart32 := int32(rangeStart64)
		rangeEnd64, _ := strconv.ParseInt(cmd[2], 10, 64)
		rangeEnd32 := int32(rangeEnd64)
		cmdDetails := ParsedCmd{rangeStart32, rangeEnd32, cmd[3]}

		switch cmd[0] {
		case "ADD":
			addRangeToGroup(c, cmdDetails)
		case "DEL":
			delRangeFromGroup(c, cmdDetails)
		case "FIND":
			result := "FINDing\n"
			c.Write([]byte(string(result)))
		case "STOP":
			fmt.Printf("Stopped Serving %s\n", c.RemoteAddr().String())
			break
		default:
			result := "ERROR: invalid cmd\n"
			c.Write([]byte(string(result)))
		}

	}
	// c.Close()
}

func addRangeToGroup(c net.Conn, cmdDetails ParsedCmd) {
	watchRange := makeRange(cmdDetails.rangeStart, cmdDetails.rangeEnd)
	//watchRange := []int32{cmdDetails.rangeStart, cmdDetails.rangeEnd
	if len(groups) == 0 {
		newGroup := Group{cmdDetails.groupName, watchRange}
		groups = append(groups, newGroup)
		fmt.Println("first group ", newGroup)
	} else {
		for i, group := range groups {
			if group.name == cmdDetails.groupName {
				fmt.Println("matched ", group)
				joined := append(group.watching, watchRange...)
				unique := makeUnique(joined)
				sort.Slice(unique, func(i, j int) bool { return unique[i] < unique[j] })
				groups[i].watching = unique
				fmt.Println("updated ", groups[i])
			} else {
				newGroup := Group{cmdDetails.groupName, watchRange}
				groups = append(groups, newGroup)
				fmt.Println("no match, creating group")
			}
		}
	}
	c.Write([]byte("OK\n"))
}

func delRangeFromGroup(c net.Conn, cmdDetails ParsedCmd) {
	delRange := makeRange(cmdDetails.rangeStart, cmdDetails.rangeEnd)
	for i, group := range groups {
		if group.name == cmdDetails.groupName {
			fmt.Println("matched ", group)
			for j := 0; j < len(groups[i].watching); j++ {
				for _, del := range delRange {
					if del == groups[i].watching[j] {
						fmt.Println(groups[i].watching[j])
						groups[i].watching = remove(groups[i].watching, j)
					}
				}
			}
			fmt.Println("delRange: ", delRange)
			fmt.Println("processed: ", groups[i])
		} else {
			fmt.Println("ERROR: no such group name")
		}
	}
	c.Write([]byte("OK\n"))
}

func makeRange(min, max int32) []int32 {
	numSlice := make([]int32, max-min+1)
	for i := range numSlice {
		numSlice[i] = min + int32(i)
	}
	return numSlice
}

func makeUnique(intSlice []int32) []int32 {
	keys := make(map[int32]bool)
	list := []int32{}
	for _, entry := range intSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

// func remove(s []int32, i int) []int32 {
// 	s[i] = s[len(s)-1]
// 	return s[:len(s)-1]
// }

// less efficient.
func remove(slice []int32, s int) []int32 {
	apdWatch := append(slice[:s], slice[s+1:]...)
	sort.Slice(apdWatch, func(i, j int) bool { return apdWatch[i] < apdWatch[j] })
	return apdWatch
}
