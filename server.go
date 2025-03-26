package main

import (
	"fmt"
	"net"
	"strings"
)

// StartServer starts the Redis server with a TCP Listener on the port 6379
// and listens for incoming connections and allocates a goroutine for each connection client.
func StartServer() {
	fmt.Println("Redis-like server starting on port 6379...")

	// listening to the port 6379
	listener, err := net.Listen("tcp", ":6379")
	if err != nil {
		fmt.Println("Error starting server:", err)
		return
	}
	// we will close the listener when the function ensd
	defer listener.Close()

	// Accepting all incoming connections and allocating them to a goroutine since they are green threads
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}

		// Handle the connection
		go handleClient(conn)
	}
}

// handleClient reads the incoming request from the client and processes it
func handleClient(conn net.Conn) {

	// we need to close the connection too offter the work of handlign with the client is done
	defer conn.Close()

	// We need to parse the request since we are using a RESP protocol to communicate with the client

	for {
		resp := NewResp(conn) // creating a new RESP instnacce to read the request
		value, err := resp.Read()
		if err != nil { // This error might be possible when there's nothing to read from the connection
			if err.Error() == "EOF" ||
				strings.Contains(err.Error(), "connection reset") ||
				strings.Contains(err.Error(), "wsarecv") {
				fmt.Println("Client disconnected")
				return
			}

			fmt.Println("Error reading request:", err)
			return
		}

		// processing the command and getting the response
		response := processCommand(value)

		// writing the response back to client
		writer := NewWriter(conn)
		writer.Write(response)
	}
}
