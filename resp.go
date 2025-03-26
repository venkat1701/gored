package main

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
)

const (
	STRING  = '+'
	ERROR   = '-'
	INTEGER = ':'
	BULK    = '$'
	ARRAY   = '*'
)

// Struct to use in the serialization and deserialization process, which will hold all the commands and args we receive
// from the client
type Value struct {
	typ   string  // determines the datatype carried by the value
	str   string  // holds the value of the string received from simple strings
	num   int     // holds the value of the integer received from integers
	bulk  string  // holds the value of the bulk string received from bulk strings
	array []Value // holds the value of the array received from arrays
}

type Resp struct {
	reader *bufio.Reader
}

func NewResp(rd io.Reader) *Resp {
	return &Resp{reader: bufio.NewReader(rd)}
}

// Now we need two methods:
// 1. to read the lines fro mthe buffer
// 2. to read the integer from the buffer

// We read one byte at a time until we reach '\r' which indicates end of line(CRLF)
// then we return the line wtihoutt eh last 2 bytes which iis the CRLF character (\r\n)
func (r *Resp) readLine() (string, error) {
	line, err := r.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return line[:len(line)-2], nil
}

// Similarly for reading integer, we make use of ParseInt to convert from the byte array
// to a 64 bit integer and returns it wrapped as int
func (r *Resp) readInteger() (int, error) {
	line, err := r.readLine()
	if err != nil {
		return 0, err
	}
	num, err := strconv.Atoi(line)
	if err != nil {
		return 0, err
	}
	return num, nil
}

// Now it is importnat to create a method that would recursively read from the buffer
// We need to read the value again for each step of the input we receive, so that we can parse it according to the character at the beginning of the line
func (r *Resp) Read() (Value, error) {
	_type, err := r.reader.ReadByte()
	if err != nil {
		return Value{}, err
	}

	switch _type {
	case STRING:
		return r.readSimpleString()
	case ERROR:
		return r.readError()
	case INTEGER:
		return r.readIntegerValue()
	case ARRAY:
		return r.readArray()
	case BULK:
		return r.readBulk()
	default:
		return Value{}, fmt.Errorf("unknown RESP type: %q", _type)
	}
}

func (r *Resp) readSimpleString() (Value, error) {
	line, err := r.readLine()
	if err != nil {
		return Value{}, err
	}
	return Value{typ: "string", str: line}, nil
}

func (r *Resp) readError() (Value, error) {
	line, err := r.readLine()
	if err != nil {
		return Value{}, err
	}
	return Value{typ: "error", str: line}, nil
}

func (r *Resp) readIntegerValue() (Value, error) {
	num, err := r.readInteger()
	if err != nil {
		return Value{}, err
	}
	return Value{typ: "integer", num: num}, nil
}

// Similarly for reading the Bulk:
// 1. Skip the first byte
// 2. Read the integer that represents the number of bytes in the bulk string
// 3. Read the bulk string followed by CRLF that indicates the end of bulk string
// 4. Return the Value object
// Reads a bulk string (`$5\r\nhello\r\n`).
func (r *Resp) readBulk() (Value, error) {
	length, err := r.readInteger()
	if err != nil {
		return Value{}, err
	}

	if length == -1 {
		return Value{typ: "null"}, nil
	}

	bulk := make([]byte, length)
	_, err = io.ReadFull(r.reader, bulk)
	if err != nil {
		return Value{}, err
	}

	// We also need to read the CRLF character that follows each bulk string.
	// If not done, the pointer will be left at \r and the Read method won't be able to
	// read the next bulk string correctly
	r.readLine()

	strBulk := string(bulk)
	return Value{typ: "bulk", bulk: strBulk}, nil
}

// Now to write something to read the array, we need to do this:
// 1. Skip the first byte as we have already read that in the Read method
// 2. Read the integer that represents the number of elements in the array
// 3. Iterate over the array and for each line, we need to call the Read method to parse
// the type according to the character at the beginnning of the line
// 4. With each loop, we append the parsed value to the array in teh Value object and return it
// An example of an array is `*2\r\n:1\r\n:2\r\n` which represents an array of two integers
func (r *Resp) readArray() (Value, error) {
	length, err := r.readInteger()
	if err != nil {
		return Value{}, err
	}

	if length == -1 {
		return Value{typ: "null"}, nil
	}

	array := make([]Value, length)
	for i := 0; i < length; i++ {
		val, err := r.Read()
		if err != nil {
			return Value{}, err
		}
		array[i] = val
	}

	return Value{typ: "array", array: array}, nil
}
