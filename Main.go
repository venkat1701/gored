package main

import (
	"fmt"
	"net"
)

func main() {
	fmt.Println("Hello World")

	// Creating a new Server
	l, err := net.Listen("tcp", ":6379")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer l.Close()

	// Listen for connections
	conn, err := l.Accept()
	if err != nil {
		fmt.Println(err)
		return
	}

	defer conn.Close()

	for {
		resp := NewResp(conn)
		value, err := resp.Read()
		if err != nil {
			fmt.Println(err)
			return
		}

		_ = value
		writer := NewWriter(conn)
		writer.Write(Value{typ: "string", str: "OK"})
	}
}
