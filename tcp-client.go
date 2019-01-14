package main

import (
	"bufio"   // Buffered I/O package to read/write to our tcp connection
	"fmt"     // Formatted I/O package for printing to the connection or console
	"log"     // Logging package to handle errors more cleanly
	"net"     // Network I/O package for our TCP/IP connection
	"os"      // OS package used for exiting when errors occur and reading lines during execution
	"strconv" // Conversion package to and from strings used to verify given arguments are integers
	"strings" // Package to manipulate UTF-8 encoded strings
	"time"    // Package for measuring and displaying time (used in the net.DialTimeout call)
)

var quit = false                                     // Boolean to track disconnecting
var defaultIP = "127.0.0.1"                          // This computer
var defaultPort = "8081"                             // Default port
var filename = fmt.Sprintf("TCCHAT_%d", os.Getpid()) // Unique filename for each client
var file, _ = os.Create(filename + ".txt")           // Text file to display the Chat
var logger = bufio.NewWriter(file)                   // Logger used to write to "TCChat.txt"

// Error checking that uses the log package to print the error and exit with status
// code 1
func errorCheck(e error) {
	if e != nil {
		log.Fatalln(e)
	}
}

func writeToFile(message string) {
	_, err := logger.WriteString(message)
	errorCheck(err)
	logger.Flush()
}

// Main function (executed first)
func main() {
	args := os.Args
	// switch to check the arguments given.
	// 1 argument means an IP address
	// 2 arguments means an IP address and a port number
	// else use localhost/127.0.0.1 and port 8081
	switch len(args) {
	case 2:
		serverAddress := args[1]
		runConn(serverAddress, defaultPort)
	case 3:
		serverAddress := args[1]
		serverPort := args[2]
		runConn(serverAddress, serverPort)
	default:
		runConn(defaultIP, defaultPort)
	}
}

// Will connect to the given IP and port if they are conform to what is expected (w.x.y.z and abcd)
// else connect to the default server address
func runConn(servAddr string, servPort string) {
	if checkServerAddress(servAddr) && checkServerPort(servPort) {
		setupChat(servAddr + ":" + servPort)
	} else {
		fmt.Println("Attempting connection to " + defaultIP + ":" + defaultPort)
		setupChat(defaultIP + ":" + defaultPort)
	}
}

// Checks if the given string is a positive integer
func checkServerPort(s string) bool {
	i, err := strconv.ParseUint(s, 10, 0)
	if err == nil && i > 0 {
		return true
	}
	fmt.Println("Port number should be a positive integer")
	return false
}

// Checks if the given string is of the form w.x.y.z and that w, x, y and z are between 1 and 254
func checkServerAddress(s string) bool {
	address := strings.Split(s, ".")
	if len(address) != 4 {
		fmt.Println("IP address should contain 4 numbers seperated by '.'")
		return false
	}
	temp := []int{} // Empty int array
	for _, i := range address {
		j, err := strconv.Atoi(i) // Convert string to int and catch potential errors
		if err != nil {
			panic(err)
		}
		// Check that the integer is between 1 and 254 (else it's not a correct value)
		if j < 0 || j > 255 {
			fmt.Println("IP address should contain numbers between 1 and 254")
			return false
		}
		temp = append(temp, j) // Add the converted integer to the array
	}
	return true
}

// Start a connection with a given server address
func setupChat(servAddress string) {
	// 3 second timeout in case the connection is slow or there is an error
	conn, err := net.DialTimeout("tcp", servAddress, time.Duration(3*time.Second))

	// Log the error if any
	if err != nil {
		fmt.Println("Connection Error: ", err.Error())
		return
	}

	// Run the client on the established connection
	runClient(conn)

	// goroutine to read messages sent from the server
	go func() {
		for !quit {
			message, err := bufio.NewReader(conn).ReadString('\n')
			if err != nil { // If there is an error whilst reading the connection, close it
				fmt.Println("Reader error: ", err)
				if e := conn.Close(); e == nil {
					fmt.Println("Closing connection...")
					quit = true
					conn.Close()
					fmt.Println("Connection closed")
				}
				break
			}
			protocolToMessage(conn, message) // Translate TCCHAT protocol messages to strings
		}
	}()

	// While loop to send messages to the server
	for !quit {
		messageToProtocol(conn) // Translate strings to TCCHAT protocol messages
	}
}

// Send specific information over the established connection 'conn'
func runClient(conn net.Conn) {
	// Get the users desired username
	writeToFile("Enter username: ")
	fmt.Print("Enter username: ")
	// Scanner allows for usernames to conatain spaces
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		username := scanner.Text()
		// Send the given username to the server
		writeToFile(username + "\r\n")
		fmt.Fprintf(conn, "TCCHAT_REGISTER\t"+username+"\n")
	}

	// Read the response from the server (We know from the protocol that it will be the server name)
	mess, _ := bufio.NewReader(conn).ReadString('\n')

	// Format the received string to get the server name
	mess = fmt.Sprintf("%s\t", mess)
	servName := strings.Split(mess, "\t")

	// Print welcome message
	writeToFile("Welcome to: " + strings.Replace(servName[1], "\n", "", -1) + "\r\n")
	fmt.Println("Welcome to: " + strings.Replace(servName[1], "\n", "", -1))
}

// Take a string of the form "TCCHAT_XXX..." and display the appropriate message
func protocolToMessage(conn net.Conn, s string) {
	// Replace "\n" in s by "" as many times as necessary (-1)
	// if -1 was 'n', it would replace "\n" only 'n' times, no more
	s = strings.Replace(s, "\n", "", -1)
	message := strings.Split(s, "\t") // Split s into an array of strings by "\t"
	
	if len(message) > 3 { // If other users send a tab, it will be split so concatenate them
		temp := ""
		for i := 2; i < len(message); i++ {
			temp += message[i]
			temp += "\t"
		}
		message[2] = temp
	} 
	// Check which protocol message was received and display accordingly
	switch message[0] {
	case "TCCHAT_USERIN":
		writeToFile(message[1] + " has joined the chat." + "\r\n")
		fmt.Println(message[1] + " has joined the chat.")
	case "TCCHAT_USEROUT":
		writeToFile(message[1] + " has left the chat." + "\r\n")
		fmt.Println(message[1] + " has left the chat.")
	case "TCCHAT_BCAST":
		writeToFile(message[1] + ": " + message[2] + "\r\n")
		fmt.Println(message[1] + ": " + message[2])
	default: // Faulty connection so should be terminated
		if err := conn.Close(); err == nil {
			quit = true
			conn.Close()
		}
	}
}

// Take input message and parse it to TCCHAT protocol
func messageToProtocol(conn net.Conn) {
	reader := bufio.NewReader(os.Stdin)
	cliMess, _ := reader.ReadString('\n')
	cliMess = strings.TrimRight(cliMess, "\r\n")
	
	// Check the message contains at most 140 characters
	if len(cliMess) > 140 {
		fmt.Println("Message should contain at most 140 characters")
	} else if strings.Contains(cliMess, "\n") {
		fmt.Println("String cannot contain a \n character")
	} else {
		writeToFile("Me: " + cliMess + "\r\n")
		// If the user types !q, disconnect him, else send the message
		if cliMess == "!q" {
			quit = true // Set quit to true to stop the loops
			writeToFile("Leaving the chat..." + "\r\n")
			fmt.Println("Leaving the chat...")
			disconnect(conn)
		} else {
			fmt.Fprintf(conn, "TCCHAT_MESSAGE\t"+cliMess+"\n")
		}
	}
}

// Disconnect the client and close the connection
func disconnect(conn net.Conn) {
	_, err := fmt.Fprintf(conn, "TCCHAT_DISCONNECT\n")
	if err != nil {
		fmt.Println(err)
	}
	conn.Close()
	writeToFile("Succesfully left the chat.\r\n")
	fmt.Print("Succesfully left the chat.")
	os.Exit(0) // Clean exit of the code
}
