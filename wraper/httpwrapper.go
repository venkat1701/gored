package main

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"sync"
)

// this is a simple http wrapper over the RESP redis server. since my redis server is running on RESP protocol, a proxy server is enough to
// handle the requests.

const respServerAddr = "127.0.0.1:6379"

// we use sync pool to manage the connections to the redis server. this is a good practice to reuse the connections
// and avoid the overhead of creating a new connection for each request
// the pool will create a new connection if there are no available connections in the pool
var clientPool = sync.Pool{
	// new function is called when the pool is empty and a new connection is needed
	New: func() any {

		// this is where we start the connection to the redis server
		conn, err := net.Dial("tcp", respServerAddr)
		if err != nil {
			panic(fmt.Sprintf("Failed to connect to RESP server: %v", err))
		}

		return conn
	},
}

// formats commands into correct RESP Bulk String format
func formatRESPCommand(args ...string) string {
	command := fmt.Sprintf("*%d\r\n", len(args)) // we start with an array format
	for _, arg := range args {
		command += fmt.Sprintf("$%d\r\n%s\r\n", len(arg), arg)
	}
	return command
}

// responsible for sending commands to the RESP server and receiving the response
// it uses the connection pool to get a connection and send the command
func sendRESPCommand(args ...string) (string, error) {
	conn := clientPool.Get().(net.Conn)
	defer clientPool.Put(conn)

	command := formatRESPCommand(args...)
	_, err := conn.Write([]byte(command))
	if err != nil {
		return "", err
	}

	reader := bufio.NewReader(conn)
	resp, err := reader.ReadString('\n') // read first line (could be $length or an error)
	if err != nil {
		return "", err
	}

	// handle bulk string response (starts with '$')
	if len(resp) > 0 && resp[0] == '$' {
		// read actual value after the length prefix
		value, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		return value[:len(value)-2], nil
	}

	return resp, nil
}

// HTTP Handler: PUT (SET key value)
func putHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

	key := r.URL.Query().Get("key")
	value := r.URL.Query().Get("value")
	if key == "" || value == "" {
		http.Error(w, "Missing key or value", http.StatusBadRequest)
		return
	}

	resp, err := sendRESPCommand("SET", key, value)
	if err != nil {
		http.Error(w, "Failed to store key", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(resp))
}

// HTTP Handler: GET (GET key)
func getHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "Missing key", http.StatusBadRequest)
		return
	}

	resp, err := sendRESPCommand("GET", key)
	if err != nil {
		http.Error(w, "Failed to retrieve key", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(resp))
}

func main() {
	http.HandleFunc("/put", putHandler)
	http.HandleFunc("/get", getHandler)

	fmt.Println("HTTP server listening on port 7171...")
	http.ListenAndServe(":7171", nil)
}
