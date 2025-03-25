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
	INTEGER = ":"
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
func (r *Resp) readLine() (line []byte, n int, err error) {
	for {
		b, err := r.reader.ReadByte()
		if err != nil {
			return nil, 0, err
		}

		n += 1
		line = append(line, b)
		if len(line) >= 2 && line[len(line)-2] == '\r' {
			break
		}
	}
	return line[:len(line)-2], n, nil
}

// Similarly for reading integer, we make use of ParseInt to convert from the byte array
// to a 64 bit integer and returns it wrapped as int
func (r *Resp) readInteger() (x int, n int, err error) {
	line, n, err := r.readLine()
	if err != nil {
		return 0, 0, err
	}

	i64, err := strconv.ParseInt(string(line), 10, 64)
	if err != nil {
		return 0, n, err
	}

	return int(i64), n, nil
}

// Now it is importnat to create a method that would recursively read from the buffer
// We need to read the value again for each step of the input we receive, so that we can parse it according to the character at the beginning of the line
func (r *Resp) Read() (Value, error) {
	_type, err := r.reader.ReadByte()

	if err != nil {
		return Value{}, err
	}

	switch _type {
	case ARRAY:
		return r.readArray()
	case BULK:
		return r.readBulk()
	default:
		fmt.Printf("Unknown type: %v", string(_type))
		return Value{}, nil
	}
}

// Now to write something to read the array, we need to do this:
// 1. Skip the first byte as we have already read that in the Read method
// 2. Read the integer that represents the number of elements in the array
// 3. Iterate over the array and for each line, we need to call the Read method to parse
// the type according to the character at the beginnning of the line
// 4. With each loop, we append the parsed value to the array in teh Value object and return it
func (r *Resp) readArray() (Value, error) {
	v := Value{}
	v.typ = "array"

	//reading the length of the array
	length, _, err := r.readInteger()
	if err != nil {
		return v, err
	}

	v.array = make([]Value, length)
	for i := 0; i < length; i++ {
		val, err := r.Read()
		if err != nil {
			return v, err
		}

		v.array[i] = val
	}

	return v, nil
}

// Similarly for reading the Bulk:
// 1. Skip the first byte
// 2. Read the integer that represents the number of bytes in the bulk string
// 3. Read the bulk string followed by CRLF that indicates the end of bulk string
// 4. Return the Value object

func (r *Resp) readBulk() (Value, error) {
	v := Value{}
	v.typ = "bulk"

	len, _, err := r.readInteger()
	if err != nil {
		return v, err
	}

	bulk := make([]byte, len)
	r.reader.Read(bulk)
	v.bulk = string(bulk)

	// We also need to read the CRLF character that follows each bulk string.
	// If not done, the pointer will be left at \r and the Read method won't be able to
	// read the next bulk string correctly
	r.readLine()

	return v, nil
}

// Now we ened to write the Marshal, that will convert the Value to bytes representing the RESP response
func (v Value) Marshal() []byte {
	switch v.typ {
	case "array":
		return v.marshalArray()
	case "bulk":
		return v.marshalBulk()
	case "string":
		return v.marshalString()
	case "null":
		return v.marshalNull()
	case "error":
		return v.marshalError()
	default:
		return []byte{}
	}
}

// for simple strings we create a byte array and add the String, follow by CRLF
// withotu CRLF, the client won't be able to read the response correctly
func (v Value) marshalString() []byte {
	var bytes []byte
	bytes = append(bytes, STRING)
	bytes = append(bytes, v.str...)
	bytes = append(bytes, '\r', '\n')
	return bytes
}

// for bulk string
func (v Value) marshalBulk() []byte {
	var bytes []byte
	bytes = append(bytes, BULK)
	bytes = append(bytes, []byte(strconv.Itoa(len(v.bulk)))...)
	bytes = append(bytes, '\r', '\n')
	bytes = append(bytes, v.bulk...)
	bytes = append(bytes, '\r', '\n')
	return bytes
}

// for arrays
func (v Value) marshalArray() []byte {
	len := len(v.array)
	var bytes []byte
	bytes = append(bytes, ARRAY)
	bytes = append(bytes, []byte(strconv.Itoa(len))...)
	bytes = append(bytes, '\r', '\n')

	for i := 0; i < len; i++ {
		bytes = append(bytes, v.array[i].Marshal()...)
	}
	return bytes
}

// for null and errors
func (v Value) marshalError() []byte {
	var bytes []byte
	bytes = append(bytes, ERROR)
	bytes = append(bytes, v.str...)
	bytes = append(bytes, '\r', '\n')

	return bytes
}

func (v Value) marshalNull() []byte {
	return []byte("$-1\r\n")
}

// Now all that we have left is to create a writer
type Writer struct {
	writer io.Writer
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{writer: w}
}

// now we need to create a method that takes Value and writes the bytes it gets from the Marshal method
func (w *Writer) Write(v Value) error {
	var bytes = v.Marshal()

	_, err := w.writer.Write(bytes)
	if err != nil {
		return err
	}

	return nil
}
