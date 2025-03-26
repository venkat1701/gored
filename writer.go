package main

import (
	"io"
	"strconv"
)

type Writer struct {
	writer io.Writer
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{writer: w}
}

func (w *Writer) Write(v Value) error {
	_, err := w.writer.Write(v.Marshal())
	return err
}

// Now we ened to write the Marshal, that will convert the Value to bytes representing the RESP response
// for simple strings we create a byte array and add the String, follow by CRLF
// without CRLF, the client won't be able to read the response correctly
func (v Value) Marshal() []byte {
	switch v.typ {
	case "string":
		return append([]byte{'+'}, append([]byte(v.str), '\r', '\n')...)
	case "error":
		return append([]byte{'-'}, append([]byte(v.str), '\r', '\n')...)
	case "integer":
		return append([]byte{':'}, append([]byte(strconv.Itoa(v.num)), '\r', '\n')...)
	case "bulk":
		if v.bulk == "" {
			return []byte("$-1\r\n")
		}
		length := strconv.Itoa(len(v.bulk))
		return []byte("$" + length + "\r\n" + v.bulk + "\r\n")
	case "array":
		length := strconv.Itoa(len(v.array))
		bytes := append([]byte{'*'}, []byte(length+"\r\n")...)
		for _, item := range v.array {
			bytes = append(bytes, item.Marshal()...)
		}
		return bytes
	case "null":
		return []byte("$-1\r\n")
	default:
		return []byte{}
	}
}
