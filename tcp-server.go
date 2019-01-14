package main

import (
	"bufio"   // Buffered I/O package to read/write to our tcp connection
	"fmt"     // Formatted I/O package for printing to the connection or console
	"log"     // Logging package to handle errors more cleanly
	"net"     // Network I/O package for our TCP/IP connection
	"os"      // OS package used for exiting when errors occur and reading lines during execution
	"strconv" // Conversion package to and from strings used to verify given arguments are integers
	"strings" // Package to manipulate UTF-8 encoded strings
	"sync"    // Syncronization package used for Mutex locks
)

var connMap map[string]net.Conn // map usernames (string) to given connections (net.Conn)
var userMap map[net.Conn]string // Map connections (net.Conn) to given usernames (string)
var mutex sync.Mutex            // Mutex lock used during the sending and receiving of messages
var closeServer = false         // Boolean to know whether or not the server should shut down or not

// Error checking that uses the log package to print the error and exit with status
// code 1
func errorCheck(e error) {
	if e != nil {
		log.Fatalln(e)
	}
}

// Checks if the given string is a positive integer
func checkServerPort(s string) bool {
	if _, err := strconv.ParseInt(s, 10, 0); err == nil {
		return true
	}
	return false
}

// Main function (executed first)
func main() {

	fmt.Println("Launching server...")

	connMap = make(map[string]net.Conn) // Allocate and initialise a map with no given size
	userMap = make(map[net.Conn]string) // Allocate and initialise a map with no given size

	args := os.Args

	var connPort = ""

	if len(args) == 2 && checkServerPort(args[1]) { // Verify a port number is given and check it
		connPort = args[1]
	} else { // Else use port 8081 by default
		connPort = "8081"
	}

	fmt.Print("IP address: ")
	getPreferredIPAddress() // Prints out the preferred IP address of the specific computer
	fmt.Println("Port number: " + connPort)

	// Listens for connection requests
	ln, err := net.Listen("tcp", ":"+connPort)

	// Error check
	if err != nil {
		fmt.Println(err)
		return
	}

	// Defer (wait till surrounding functions have finished) the execution of ln.Close()
	defer ln.Close()

	// Semi-infinite loop that accepts connections, checks for errors and executes a goroutine
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Accept error: ", err)
			return
		}
		go connection(conn) // goroutine execution of the connection function concurrently
	}
}

// UDP doesn't establish a connection and the destination doesn't need to exist. The function gets
// the local IP address if it were to connect to that target address.
// conn.LocalAddr().(*net.UPDAddr) get the preferred (obviously) outbound IP address
func getPreferredIPAddress() {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	os.Stdout.WriteString(localAddr.IP.String() + "\n") // Prints the IP address as a string
}

//Connection function giving the machines to which it is connected (1), reading the messages
// it receives via net.Conn, standard error check, string parsing from the client message,
// mutex locking to handle the received message and unlocking afterwards.
func connection(c net.Conn) {
	fmt.Printf("Serving %s\n", c.RemoteAddr().String()) // (1)

	for {
		cliMess, err := bufio.NewReader(c).ReadString('\n')
		if err != nil {
			fmt.Println("Connection error: ", err)
			break
		}

		message := fmt.Sprintf("%s", cliMess)
		mutex.Lock()
		protocolToMessage(c, message)
		mutex.Unlock()
	}
	
	fmt.Println("Closing connection...")
	var temp = userMap[c] // Temporary copy of the username associated to *c in userMap
	delete(userMap, c)    // Delete the username associated to *c in userMap
	delete(connMap, temp) // Delete the connection associated to the previously deleted username in connMap
	c.Close()             // Connection closing if the for loop is exited
	fmt.Println("Connection closed.")
}

// Convert TCCHAT protocol to actual messages
func protocolToMessage(c net.Conn, s string) {
	fmt.Print("Recieved: " + s)       // Used to verify what was being received
	message := strings.Split(s, "\t") // Split the received message into an array of strings by \t
	username := "" // Empty string that will contain the specified username
	registerCount := 0 // Counter for the amount of times TCCHAT_REGISTER is received
	
	if len(message) > 1 { // If message has only one string in it, it's necessarily a disconnect call
		// Replace "\n" in message by "" as many times as necessary (-1)
		// if -1 was 'n', it would replace "\n" only 'n' times, no more
		username = strings.Replace(message[1], "\n", "", -1)
	}
	
	// Check if the connection has only sent 1 TCCHAT_REGISTER
	if registerCount > 1 {
		fmt.Println("Corrupted connection detected !")
		c.Close()
		fmt.Println("Connection closed")
	}
	// Prettier if else if loop checking the contents of message[0] which contains the prefix
	// of the protocol message
	switch message[0] {
	case "TCCHAT_REGISTER": // A new user has joined the server
		registerUser(c, username)
		registerCount += 1 // Increment counter by 1
		
	case "TCCHAT_MESSAGE": // A message has been received from a connected client
		sendMessageAll(c, message[1])

	case "TCCHAT_DISCONNECT\n": // In case of a disconnect, the \n will still be part of the message
		userDisconnect(userMap[c])
		
	default: // Message received is not of the correct form, close the connection
		if err := c.Close(); err == nil {
			c.Close()
		}
	}
}

// Add the new user and linked connection to both the userMap and connMap
func registerUser(conn net.Conn, username string) {
	// Check if the username has already been used
	newUsername := checkDuplicateUsername(username)
	// Notify other users a new user has joined
	userConnect(newUsername) // Called before adding the new user to the maps to prevent sending a 'joined' notification to himself (useless)
	connMap[newUsername] = conn
	userMap[conn] = newUsername
	// fmt.Println(username + " joined the chat.") // Used to verify the reception of the username
	conn.Write([]byte("TCCHAT_WELCOME\tTCChat G7\n")) // Send welcome to the new user
}

// Change username to usernameN (N an integer) if username has already established a connection
func checkDuplicateUsername(username string) string {
	// Count the times the username exists
	occurencesOfUsername := 0
	for _, value := range userMap {
		if value == username {
			occurencesOfUsername += 1
		}
	}
	// Add the amount of occurences to the end of the username
	if occurencesOfUsername > 0 {
		username = username + strconv.Itoa(occurencesOfUsername+1)
	}
	// Return it
	return username
}

// Range through all the connected users and notify them *username has joined
func userConnect(username string) {
	for key := range connMap {
		connMap[key].Write([]byte("TCCHAT_USERIN\t" + username + "\n"))
	}
}

// Range through all the connected users and notify them *username has left
func userDisconnect(username string) {
	for key := range connMap {
		connMap[key].Write([]byte("TCCHAT_USEROUT\t" + username + "\n"))
	}

	var temp = connMap[username] // Temporary copy of the connection associated to *username in connMap
	delete(connMap, username)    // Delete the connection associated to *username in connMap
	delete(userMap, temp)        // Delete the username associated to the previously deleted connection in userMap
	serverShutdown(temp)         // Check whether or not to shutdown the server
}

// Broadcast received messages to all clients except the client that sent the message (useless)
func sendMessageAll(conn net.Conn, mess string) {
	for key := range connMap {
		if connMap[key] != conn {
			connMap[key].Write([]byte("TCCHAT_BCAST\t" + userMap[conn] + "\t" + mess + "\n"))
		}
	}
}

// Function to verify the conditions to safely shutdown the server
func serverShutdown(finalConn net.Conn) {
	var shutdown string
	// If all clients have left... (Note this is never called when the server is started because no client has disconnected even though there are no clients)
	if len(connMap) == 0 {
		if err := finalConn.Close(); err == nil {
			fmt.Println("Closing final connection...") // Close the connection of the last client
			finalConn.Close()
		}
		fmt.Println("All users have left the chat")
		fmt.Print("Do you wish to shut down the server ? yes/no : ")
		fmt.Scanln(&shutdown) // Put the message typed into shutdown
		if shutdown == "yes" {
			fmt.Print("Shutting down server...")
			os.Exit(0) // Server shutdown with no error
		}
	}
}
