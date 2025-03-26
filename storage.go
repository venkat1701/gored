package main

import (
	"fmt"
	"strings"
	"sync"
)

// We use sync.RWMutex coz our server is supposed to handle requests concurrently.
// We use RWMutex to ensure that the SETs map is not modified by multiple threads at same time
// Using HashMap to store the key value pairs
var SETs = map[string]string{}
var SETsMutex = sync.RWMutex{}

func processCommand(value Value) Value {
	if value.typ != "array" {
		return Value{typ: "error", str: "ERR invalid command format"}
	}

	// we need to make sure if the command is present or not
	if len(value.array) == 0 {
		return Value{typ: "error", str: "ERR empty command"}
	}

	// here's the point where we extract the command form the array
	cmdValue := value.array[0]

	// making sure the command is of the right type
	if cmdValue.typ == "" || (cmdValue.typ != "bulk" && cmdValue.typ != "string") {
		return Value{typ: "error", str: "ERR invalid command type"}
	}

	// converting the command to uppercase
	var cmd string
	if cmdValue.typ == "bulk" {
		cmd = strings.ToUpper(cmdValue.bulk)
	} else {
		cmd = strings.ToUpper(cmdValue.str)
	}

	// debugging the command
	fmt.Printf("Received command: %s, Total args: %d\n", cmd, len(value.array))

	// Command processing
	switch cmd {
	case "PING":
		// ping can have 0 or 1 argument only
		if len(value.array) > 2 {
			return Value{typ: "error", str: "ERR wrong number of arguments for 'PING' command"}
		}

		// if there's no args, returning PONG
		if len(value.array) == 1 {
			return Value{typ: "string", str: "PONG"}
		}

		// if an argument is provided return that argument
		arg := value.array[1]
		if arg.typ == "bulk" {
			return Value{typ: "bulk", bulk: arg.bulk}
		}
		return Value{typ: "string", str: arg.str}

	case "SET":
		if len(value.array) != 3 {
			return Value{typ: "error", str: "ERR wrong number of arguments for 'SET' command"}
		}

		if value.array[1].bulk == "" || value.array[2].bulk == "" {
			return Value{typ: "error", str: "ERR invalid key or value"}
		}

		key := value.array[1].bulk
		val := value.array[2].bulk

		// Acquiring the lock before writing the value to the map
		// This is to ensure that only one thread can write to the map at a time just the way it happens in Redis
		SETsMutex.Lock()
		SETs[key] = val
		SETsMutex.Unlock()

		return Value{typ: "string", str: "OK"}

	case "GET":
		if len(value.array) != 2 {
			return Value{typ: "error", str: "ERR wrong number of arguments for 'GET' command"}
		}

		// we need to see if the key is not empty
		if value.array[1].bulk == "" {
			return Value{typ: "error", str: "ERR invalid key"}
		}

		key := value.array[1].bulk

		SETsMutex.RLock()
		val, exists := SETs[key]
		SETsMutex.RUnlock()

		if !exists {
			return Value{typ: "null"}
		}

		return Value{typ: "bulk", bulk: val}

	default:
		return Value{typ: "error", str: fmt.Sprintf("ERR unknown command '%s'", cmd)}
	}
}
