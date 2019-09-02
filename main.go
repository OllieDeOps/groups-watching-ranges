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
		// If group name is provided
		if len(cmd) == 4 {
			cmdDetails := ParsedCmd{rangeStart32, rangeEnd32, cmd[3]}
			switch cmd[0] {
			case "ADD":
				addRangeToGroup(c, cmdDetails)
				fmt.Println(groups)
			case "DEL":
				delRangeFromGroup(c, cmdDetails)
				fmt.Println(groups)
			default:
				result := "ERROR: invalid cmd\n"
				c.Write([]byte(string(result)))
			}
		} else if len(cmd) == 3 {
			cmdDetails := ParsedCmd{rangeStart: rangeStart32, rangeEnd: rangeEnd32}
			switch cmd[0] {
			case "DEL":
				delRangeFromAllGroups(c, cmdDetails)
				fmt.Println(groups)
			case "FIND":
				// findRangeForAllGroups(c, cmdDetails)
				result := "FINDing\n"
				c.Write([]byte(string(result)))
			default:
				result := "ERROR: invalid cmd\n"
				c.Write([]byte(string(result)))
			}
		} else if len(cmd) == 2 {
			// cmdDetails := ParsedCmd{rangeStart: rangeStart32}
			switch cmd[0] {
			case "FIND":
				// findValueForAllGroups(c, cmdDetails)
				result := "FINDing\n"
				c.Write([]byte(string(result)))
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
	// c.Close()
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
				sort.Slice(unique, func(i, j int) bool { return unique[i] < unique[j] })
				groups[i].watching = unique
				c.Write([]byte("OK\n"))
				return
			}
			if groups[i].name != cmdDetails.groupName {
				notMatching = append(notMatching, groups[i])
			}
		}

		fmt.Println("notMatching: ", notMatching)
		newGroup := Group{cmdDetails.groupName, watchRange}
		groups = append(notMatching, newGroup)
	}
	c.Write([]byte("OK\n"))
}

func delRangeFromGroup(c net.Conn, cmdDetails ParsedCmd) {
	delRange := makeRange(cmdDetails.rangeStart, cmdDetails.rangeEnd)
	for i, group := range groups {
		if group.name == cmdDetails.groupName {
			for j := 0; j < len(groups[i].watching); j++ {
				for _, del := range delRange {
					if del == groups[i].watching[j] {
						groups[i].watching = remove(groups[i].watching, j)
					}
				}
			}
		}
	}
	c.Write([]byte("OK\n"))
}

// Hmmm... perhaps combine the delete funcs
func delRangeFromAllGroups(c net.Conn, cmdDetails ParsedCmd) {
	delRange := makeRange(cmdDetails.rangeStart, cmdDetails.rangeEnd)
	for i := range groups {
		for d := range delRange {
			for j := 0; j < len(groups[i].watching); j++ {
				if delRange[d] == groups[i].watching[j] {

					fmt.Println("delRange[d]: ", delRange[d])
					fmt.Println("j: ", j)
					//fmt.Println("groups[i].watching[d]: ", groups[i].watching[d])
					//if d < len(groups[i].watching) {
					fmt.Println("groups[i].watching: ", groups[i].watching)
					groups[i].watching = remove(groups[i].watching, j)
					//}
				}
			}
		}
	}
	// for i := range groups {
	// 	for j := 0; j < len(groups[i].watching)-1; j++ {
	// 		for _, del := range delRange {
	// 			if del == groups[i].watching[j] {

	// 				fmt.Println("del: ", del)
	// 				fmt.Println("j: ", j)
	// 				//fmt.Println("groups[i].watching[d]: ", groups[i].watching[d])
	// 				//if d < len(groups[i].watching) {
	// 				fmt.Println("groups[i].watching: ", groups[i].watching)
	// 				groups[i].watching = remove(groups[i].watching, j)
	// 				//}
	// 			} else {
	// 				fmt.Println("d != j")
	// 			}
	// 		}
	// 	}
	// }

	c.Write([]byte("OK\n"))
}

func makeRange(min, max int32) []int32 {
	numSlice := make([]int32, max-min+1)
	for i := range numSlice {
		numSlice[i] = min + int32(i)
	}
	return numSlice
}

func makeIntRange(min, max int32) []int {
	numSlice := make([]int, max-min+1)
	for i := range numSlice {
		numSlice[i] = int(min) + int(i)
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

func remove(s []int32, j int) []int32 {
	s[j] = s[len(s)-1]
	sort.Slice(s, func(i, l int) bool { return s[i] < s[l] })
	return s[:len(s)-1]
}

// less efficient due to shifting all elements of array for each append.
// func slowRemove(slice []int32, s int) []int32 {
// 	apdWatch := append(slice[:s], slice[s+1:]...)
// 	sort.Slice(apdWatch, func(i, j int) bool { return apdWatch[i] < apdWatch[j] })
// 	return apdWatch
// }
