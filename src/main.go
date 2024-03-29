package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

const (
	connHost = "localhost"
	connPort = "3333"
	connType = "tcp"
)

// Group watches numbers
type Group struct {
	name     string
	watching []uint32
}

var groups []Group

// ParsedCmd holds type-converted data sent over network to pass around
type ParsedCmd struct {
	rangeStart uint32
	rangeEnd   uint32
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
		// Our recieved data as a string.
		netData, err := bufio.NewReader(c).ReadString('\n')
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Break up command data into slice
		cmd := strings.Fields(string(netData))
		// Check validity of command data
		rangeStartIsNumber := true
		rangeEndIsNumber := true
		if len(cmd) == 0 || len(cmd) > 4 {
			c.Write([]byte(string("ERROR: invalid command\n")))
		} else {
			if cmd[0] == "STOP" {
				fmt.Printf("Stopped Serving %s\n", c.RemoteAddr().String())
				c.Write([]byte(string("Disconnecting\n")))
				break
			}
			if cmd[0] == "ADD" || cmd[0] == "DEL" || cmd[0] == "FIND" {
				var rangeStart uint32
				var rangeEnd uint32
				var checkRangeEnd int64
				var checkRangeStart int64
				if len(cmd) > 1 {
					if _, err := strconv.Atoi(cmd[1]); err == nil {
						checkRangeStart, _ = strconv.ParseInt(cmd[1], 10, 64)
					} else {
						rangeStartIsNumber = false
					}
				}
				if len(cmd) > 2 {
					if _, err := strconv.Atoi(cmd[2]); err == nil {
						checkRangeEnd, _ = strconv.ParseInt(cmd[2], 10, 64)
					} else {
						rangeEndIsNumber = false
					}
				}
				if rangeStartIsNumber == false || rangeEndIsNumber == false || checkRangeStart < 0 || checkRangeStart > 4294967295 || checkRangeEnd < 0 || checkRangeEnd > 4294967295 {
					if rangeStartIsNumber == false || rangeEndIsNumber == false {
						c.Write([]byte(string("ERROR: args given for range must be a number\n")))
					} else {
						c.Write([]byte(string("ERROR: given value out of range\n")))
					}
				} else if len(cmd) > 2 && checkRangeStart > checkRangeEnd {
					c.Write([]byte(string("ERROR: first value argument must be a number smaller than the second\n")))
				} else {
					// Route command args to proper function
					var rangeStart64 int64
					var rangeEnd64 int64
					var cmdDetails ParsedCmd
					if len(cmd) == 4 {
						rangeStart64, _ = strconv.ParseInt(cmd[1], 10, 64)
						rangeEnd64, _ = strconv.ParseInt(cmd[2], 10, 64)
						rangeStart = uint32(rangeStart64)
						rangeEnd = uint32(rangeEnd64)
						cmdDetails = ParsedCmd{rangeStart, rangeEnd, cmd[3]}
					} else if len(cmd) == 3 {
						rangeStart64, _ = strconv.ParseInt(cmd[1], 10, 64)
						rangeEnd64, _ = strconv.ParseInt(cmd[2], 10, 64)
						rangeStart = uint32(rangeStart64)
						rangeEnd = uint32(rangeEnd64)
						cmdDetails = ParsedCmd{rangeStart: rangeStart, rangeEnd: rangeEnd}
					} else {
						rangeStart64, _ = strconv.ParseInt(cmd[1], 10, 64)
						rangeStart = uint32(rangeStart64)
						cmdDetails = ParsedCmd{rangeStart: rangeStart}
					}
					if cmd[0] == "ADD" && len(cmd) == 4 {
						fmt.Println("adding range...")
						addRangeToGroup(c, cmdDetails)
						fmt.Println("done")
					} else if cmd[0] == "DEL" {
						fmt.Println("deleting range...")
						delRange(c, cmdDetails)
						fmt.Println("done")
					} else if cmd[0] == "FIND" {
						fmt.Println("finding groups...")
						findWatchingGroups(c, cmdDetails)
						fmt.Println("done")
					}
				}
			} else {
				c.Write([]byte(string("ERROR: invalid command\n")))
			}
		}
	}
}

func addRangeToGroup(c net.Conn, cmdDetails ParsedCmd) {
	notMatching := make([]Group, 0)
	watchRange := makeRange(cmdDetails.rangeStart, cmdDetails.rangeEnd)
	if len(groups) == 0 {
		newGroup := Group{cmdDetails.groupName, watchRange}
		groups = append(groups, newGroup)
	} else {
		for i := range groups {
			if groups[i].name == cmdDetails.groupName {
				joined := append(groups[i].watching, watchRange...)
				unique := makeIntsUnique(joined)
				groups[i].watching = unique
				c.Write([]byte("OK\n"))
				return
			}
			if groups[i].name != cmdDetails.groupName {
				notMatching = append(notMatching, groups[i])
			}
		}
		newGroup := Group{cmdDetails.groupName, watchRange}
		groups = append(notMatching, newGroup)
	}
	c.Write([]byte("OK\n"))
}

func delRange(c net.Conn, cmdDetails ParsedCmd) {
	delRange := makeRange(cmdDetails.rangeStart, cmdDetails.rangeEnd)
	if cmdDetails.groupName != "" {
		for i, group := range groups {
			if group.name == cmdDetails.groupName {
				for d := range delRange {
					for j := 0; j < len(groups[i].watching); j++ {
						if delRange[d] == groups[i].watching[j] {
							groups[i].watching = remove(groups[i].watching, j)
						}
					}
				}
			}
		}
	} else {
		for i := range groups {
			for d := range delRange {
				for j := 0; j < len(groups[i].watching); j++ {
					if delRange[d] == groups[i].watching[j] {
						groups[i].watching = remove(groups[i].watching, j)
					}
				}
			}
		}
	}
	c.Write([]byte("OK\n"))
}

func findWatchingGroups(c net.Conn, cmdDetails ParsedCmd) {
	foundGroups := []string{}
	if cmdDetails.rangeEnd != 0 {
		findRange := makeRange(cmdDetails.rangeStart, cmdDetails.rangeEnd)
		for i := range groups {
			for f := range findRange {
				for w := range groups[i].watching {
					if findRange[f] == groups[i].watching[w] {
						foundGroups = append(foundGroups, groups[i].name)
						unique := makeStringsUnique(foundGroups)
						foundGroups = unique
					}
				}
			}
		}
	} else {
		for i := range groups {
			for w := range groups[i].watching {
				if cmdDetails.rangeStart == groups[i].watching[w] {
					foundGroups = append(foundGroups, groups[i].name)
					unique := makeStringsUnique(foundGroups)
					foundGroups = unique
				}
			}
		}
	}

	if len(foundGroups) > 0 {
		stringsToBytes := "\x00" + strings.Join(foundGroups, "\x20\x00")
		c.Write([]byte(stringsToBytes + "\n"))
	} else {
		c.Write([]byte("ERROR: no results\n"))
	}
}

func makeRange(min, max uint32) []uint32 {
	numSlice := make([]uint32, max-min+1)
	for i := range numSlice {
		numSlice[i] = min + uint32(i)
	}
	return numSlice
}

func makeIntsUnique(intSlice []uint32) []uint32 {
	keys := make(map[uint32]bool)
	list := []uint32{}
	for _, entry := range intSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func makeStringsUnique(stringSlice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range stringSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func remove(s []uint32, j int) []uint32 {
	s[j] = s[len(s)-1]
	return s[:len(s)-1]
}
