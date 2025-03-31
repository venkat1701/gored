package main

import (
	"fmt"
	"runtime"
)

// This is where we start the server
func main() {
	// we check for some diagnostic information
	// like the Go version and number of CPUs available
	fmt.Printf("Starting Gored cache with Go version %s\n", runtime.Version())
	fmt.Printf("System has %d CPUs\n", runtime.NumCPU())

	// and then we start our resp server on port 7171
	StartServer()
}
