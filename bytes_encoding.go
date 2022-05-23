package hotstuff

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	BYTES_ARRAY_PREFIX_LENGTH = 4
)

// not used
func toByteArray(msg string) []byte {
	msg_byte_arr := []byte(msg)
	length := len(msg_byte_arr)
	length_byte_arr := make([]byte, BYTES_ARRAY_PREFIX_LENGTH)
	binary.BigEndian.PutUint32(length_byte_arr[0 : BYTES_ARRAY_PREFIX_LENGTH], uint32(length))
	msg_byte_arr = append(length_byte_arr, msg_byte_arr...)
	return msg_byte_arr
}

// use this method after "Message.marshall()"
func augmentByteArrayWithLength(msg_byte_arr []byte) []byte {
	length := len(msg_byte_arr)
	length_byte_arr := make([]byte, BYTES_ARRAY_PREFIX_LENGTH)
	binary.BigEndian.PutUint32(length_byte_arr[0 : BYTES_ARRAY_PREFIX_LENGTH], uint32(length))
	msg_byte_arr_augmented := append(length_byte_arr, msg_byte_arr...)
	return msg_byte_arr_augmented
}

// assumes length_byte_arr is length-4 bytes array
func getMsgLength(length_byte_arr []byte) int {
	length := int(binary.BigEndian.Uint32(length_byte_arr))
	return length
}

func toString(msg_byte_arr []byte) string {
	return string(msg_byte_arr)
}

func getBytesFromAugmented(augmented []byte) ([]byte) {
	// 1st step: extract length of message to be read (number of bytes)
	length_buf := augmented[:BYTES_ARRAY_PREFIX_LENGTH]
	rest := augmented[BYTES_ARRAY_PREFIX_LENGTH:]
	msg_length := getMsgLength(length_buf)

	// 2nd step: slice up to 'msg_length'
	msg_bytes := rest[:msg_length]
	return msg_bytes
}

func getMessageFromReader(reader io.Reader) ([]byte, error) {
	// 1st step: extract length of message to be read (number of bytes)
	length_buf := make([]byte, BYTES_ARRAY_PREFIX_LENGTH)
	_, readLengthBuf_err := io.ReadFull(reader, length_buf)
	// error handling
	if readLengthBuf_err != nil {
		fmt.Println("Receiver: failed to read first 4 bytes from reader!")
		return []byte{}, errors.New("Failed to read first 4 bytes from reader")
		// panic(readLengthBuf_err)
	}
	msg_length := getMsgLength(length_buf)
	// fmt.Println("Receiver: message length = ", msg_length)

	// 2nd step: read 'msg_length' - many bytes from reader 
	msg_buf := make([]byte, msg_length)
	_, readMsg_err := io.ReadFull(reader, msg_buf)
	// error handling
	if readMsg_err != nil {
		fmt.Println("Receiver: failed to read message from reader!")
		return []byte{}, errors.New("Failed to read message from reader")
		// panic(readMsg_err)
	}
	// fmt.Println("Receiver: message buffer = ", msg_buf)
	
	// msg := toString(msg_buf)

	return msg_buf, nil
}

func pad(data_bytes []byte, required int) []byte {
	// pad with enough "_" characters to make the length of 'data_bytes' a multiple of 'required' 
	pad_num := required - (len(data_bytes) % required)
	dummy_string := strings.Repeat("_", pad_num)
	res := append(data_bytes, []byte(dummy_string)...)
	return res
}

func split(buf []byte, lim int) [][]byte {
	// adapted from Github repo: https://gist.github.com/xlab/6e204ef96b4433a697b3
	var chunk []byte
	chunks := make([][]byte, 0, len(buf)/lim+1)
	for len(buf) >= lim {
		chunk, buf = buf[:lim], buf[lim:]
		chunks = append(chunks, chunk)
	}
	if len(buf) > 0 {
		chunks = append(chunks, buf[:])
	}
	return chunks
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
