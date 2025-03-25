package main

import "sync"

func ping(args []Value) Value {
	return Value{typ: "string", str: "PONG"}
}

var SETs = map[string]string{}
var SETsMutex = sync.RWMutex{}

// We use sync.RWMutex coz our server is supposed to handle requests concurrently.
// We use RWMutex to ensure that the SETs map is not modified by multiple threads at same time
// Using HashMap to store the key value pairs
func set(args []Value) Value {
	if len(args) != 2 {
		return Value{typ: "error", str: "ERR wrong number of arguments"}
	}

	key := args[0].bulk
	value := args[1].bulk

	SETsMutex.Lock()
	SETs[key] = value
	SETsMutex.Unlock()

	return Value{typ: "string", str: "OK"}
}

func get(args []Value) Value {
	if len(args) != 1 {
		return Value{typ: "error", str: "ERR wrong number of arguments"}
	}

	key := args[0].bulk
	SETsMutex.RLock()
	value, ok := SETs[key]
	SETsMutex.RUnlock()

	if !ok {
		return Value{typ: "null"}
	}

	return Value{typ: "bulk", bulk: value}
}

var Handlers = map[string]func([]Value) Value{
	"PING": ping,
	"SET":  set,
	"GET":  get,
}
