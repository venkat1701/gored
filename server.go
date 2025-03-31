package main

import (
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"
)

// StartServer starts the redis compatible RESP server on port 7171
// (instead of 6379, to comply with the assignment requirements)
func StartServer() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "7171"
	}

	// we want to use all the available CPU cores for the server
	// this is important for performance, especially when handling multiple connections
	runtime.GOMAXPROCS(runtime.NumCPU())

	fmt.Println("Key-Value Cache server starting on port", port, "...")
	fmt.Println("Available CPU cores:", runtime.NumCPU())

	// starting the tcp listener on port 7171
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Println("Error starting server:", err)
		return
	}

	// we'll close the listener when the function ends
	defer listener.Close()

	fmt.Println("Server ready to accept connections")

	// we accept all the connections in a loop
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}

		// each client connection gets its own goroutine
		go handleClient(conn)
	}
}

// handleClient reads the incoming requests from a client and processes them
func handleClient(conn net.Conn) {
	// making sure we close the connection when we're done
	defer conn.Close()

	// keep handling commands in a loop until client disconnects
	for {
		// creating a new RESP parser for this connection
		resp := NewResp(conn)

		// reeading the next command from client
		value, err := resp.Read()
		if err != nil {
			// handle client disconnection gracefully
			if err.Error() == "EOF" ||
				strings.Contains(err.Error(), "connection reset") ||
				strings.Contains(err.Error(), "wsarecv") {
				// Client disconnected - this is normal
				return
			}

			// If we're here, something unexpected happened
			fmt.Println("Error reading request:", err)
			return
		}

		// process the command to get a response
		response := processCommand(value)

		// write response back to client
		writer := NewWriter(conn)
		err = writer.Write(response)
		if err != nil {
			fmt.Println("Error writing response:", err)
			return
		}
	}
}
