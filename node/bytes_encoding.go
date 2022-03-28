package main

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	BYTES_ARRAY_PREFIX_LENGTH = 4
)


func toByteArray(msg string) []byte {
	msg_byte_arr := []byte(msg)
	length := len(msg_byte_arr)
	length_byte_arr := make([]byte, BYTES_ARRAY_PREFIX_LENGTH)
	binary.BigEndian.PutUint32(length_byte_arr[0 : BYTES_ARRAY_PREFIX_LENGTH], uint32(length))
	msg_byte_arr = append(length_byte_arr, msg_byte_arr...)
	return msg_byte_arr
}

// assumes length_byte_arr is length-4 bytes array
func getMsgLength(length_byte_arr []byte) int {
	length := int(binary.BigEndian.Uint32(length_byte_arr))
	return length
}

func toString(msg_byte_arr []byte) string {
	return string(msg_byte_arr)
}

func getMessageFromReader(reader *bufio.Reader) (string, error) {
	// 1st step: extract length of message to be read (number of bytes)
	length_buf := make([]byte, BYTES_ARRAY_PREFIX_LENGTH)
	_, readLengthBuf_err := io.ReadFull(reader, length_buf)
	// error handling
	if readLengthBuf_err != nil {
		fmt.Println("Receiver: failed to read first 4 bytes from reader!")
		return "", errors.New("Failed to read first 4 bytes from reader")
		// panic(readLengthBuf_err)
	}
	msg_length := getMsgLength(length_buf)
	fmt.Println("Receiver: message length = ", msg_length)

	// 2nd step: read 'msg_length' - many bytes from reader 
	msg_buf := make([]byte, msg_length)
	_, readMsg_err := io.ReadFull(reader, msg_buf)
	// error handling
	if readMsg_err != nil {
		fmt.Println("Receiver: failed to read message from reader!")
		return "", errors.New("Failed to read message from reader")
		// panic(readMsg_err)
	}
	fmt.Println("Receiver: message buffer = ", msg_buf)
	msg := toString(msg_buf)

	return msg, nil
}

// ----------------------------------------------------------------------------------------------------------


// for debugging purposes

func example_run() {
	msg := "falcon and many eagles and many eagles and many eagles"
	bytes_array := toByteArray(msg)
	// extra_msg := "extra stuff"
	// bytes_array = append(bytes_array, []byte(extra_msg)...)
	fmt.Println(bytes_array)
	fmt.Println("get message length: ", getMsgLength(bytes_array))
	// fmt.Println("back to string: ", toString(bytes_array))
	// fmt.Println("equal strings: ", msg == toString(toByteArray(msg)))
}