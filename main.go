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

// Group defines the watcher
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
		// Our recieved string.
		netData, err := bufio.NewReader(c).ReadString('\n')
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		// Parse cmd ints
		cmd := strings.Fields(string(netData))
		fmt.Println(cmd)
		rangeStart64, _ := strconv.ParseInt(cmd[1], 10, 64)
		rangeStart32 := uint32(rangeStart64)
		// If group name is provided
		if len(cmd) == 4 {
			rangeEnd64, _ := strconv.ParseInt(cmd[2], 10, 64)
			rangeEnd32 := uint32(rangeEnd64)
			cmdDetails := ParsedCmd{rangeStart32, rangeEnd32, cmd[3]}
			switch cmd[0] {
			case "ADD":
				fmt.Println("adding range...")
				addRangeToGroup(c, cmdDetails)
				fmt.Println("done")
			case "DEL":
				fmt.Println("deleting range...")
				delRangeFromGroup(c, cmdDetails)
				fmt.Println("done")
			default:
				result := "ERROR: invalid cmd\n"
				c.Write([]byte(string(result)))
			}
		} else if len(cmd) == 3 {
			rangeEnd64, _ := strconv.ParseInt(cmd[2], 10, 64)
			rangeEnd32 := uint32(rangeEnd64)
			cmdDetails := ParsedCmd{rangeStart: rangeStart32, rangeEnd: rangeEnd32}
			switch cmd[0] {
			case "DEL":
				fmt.Println("deleting range...")
				delRangeFromAllGroups(c, cmdDetails)
				fmt.Println("done")
			case "FIND":
				fmt.Println("finding groups...")
				findRangeForAllGroups(c, cmdDetails)
				fmt.Println("done")
			default:
				result := "ERROR: invalid cmd\n"
				c.Write([]byte(string(result)))
			}
		} else if len(cmd) == 2 {
			cmdDetails := ParsedCmd{rangeStart: rangeStart32}
			switch cmd[0] {
			case "FIND":
				fmt.Println("finding groups...")
				findValueForAllGroups(c, cmdDetails)
				fmt.Println("done")
			case "STOP":
				fmt.Printf("Stopped Serving %s\n", c.RemoteAddr().String())
				break
			default:
				result := "ERROR: invalid cmd\n"
				c.Write([]byte(string(result)))
			}
		} else if len(cmd) == 1 {
			switch cmd[0] {
			case "STOP":
				fmt.Printf("Stopped Serving %s\n", c.RemoteAddr().String())
				break
			default:
				result := "ERROR: invalid cmd\n"
				c.Write([]byte(string(result)))
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
				unique := makeUnique(joined)
				//sort.Slice(unique, func(i, j int) bool { return unique[i] < unique[j] })
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

// Hmmm... perhaps combine the delete functions
func delRangeFromGroup(c net.Conn, cmdDetails ParsedCmd) {
	delRange := makeRange(cmdDetails.rangeStart, cmdDetails.rangeEnd)
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
	c.Write([]byte("OK\n"))
}

// Hmmm... perhaps combine the delete functions
func delRangeFromAllGroups(c net.Conn, cmdDetails ParsedCmd) {
	delRange := makeRange(cmdDetails.rangeStart, cmdDetails.rangeEnd)
	for i := range groups {
		for d := range delRange {
			for j := 0; j < len(groups[i].watching); j++ {
				if delRange[d] == groups[i].watching[j] {
					groups[i].watching = remove(groups[i].watching, j)
				}
			}
		}
	}
	c.Write([]byte("OK\n"))
}

func findRangeForAllGroups(c net.Conn, cmdDetails ParsedCmd) {
	findRange := makeRange(cmdDetails.rangeStart, cmdDetails.rangeEnd)
	foundGroups := make([]string, 0)
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

	stringsToBytes := "\x00" + strings.Join(foundGroups, "\x20\x00")
	c.Write([]byte(stringsToBytes + "\n"))
}

func findValueForAllGroups(c net.Conn, cmdDetails ParsedCmd) {
	foundGroups := []string{}
	for i := range groups {
		for w := range groups[i].watching {
			if groups[i].watching[w] == cmdDetails.rangeStart {
				foundGroups = append(foundGroups, groups[i].name)
				unique := makeStringsUnique(foundGroups)
				foundGroups = unique
			}
		}
	}

	stringsToBytes := "\x00" + strings.Join(foundGroups, "\x20\x00")
	c.Write([]byte(stringsToBytes + "\n"))
}

func makeRange(min, max uint32) []uint32 {
	numSlice := make([]uint32, max-min+1)
	for i := range numSlice {
		numSlice[i] = min + uint32(i)
	}
	return numSlice
}

// func parseRange(wRange []uint32) []uint32 {
// 	for i := range wRange {
// 		if i+1 !=
// 	}
// }

func makeIntRange(min, max uint32) []int {
	numSlice := make([]int, max-min+1)
	for i := range numSlice {
		numSlice[i] = int(min) + int(i)
	}
	return numSlice
}

func makeUnique(intSlice []uint32) []uint32 {
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
	//sort.Slice(s, func(i, l int) bool { return s[i] < s[l] })
	return s[:len(s)-1]
}

// less efficient due to shifting all elements of array for each append.
// func slowRemove(slice []uint32, s int) []uint32 {
// 	apdWatch := append(slice[:s], slice[s+1:]...)
// 	sort.Slice(apdWatch, func(i, j int) bool { return apdWatch[i] < apdWatch[j] })
// 	return apdWatch
// }
