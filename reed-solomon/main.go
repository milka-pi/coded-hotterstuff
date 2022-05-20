package main

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/klauspost/reedsolomon"
	// "github.com/vivint/infectious"
)

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



func main() {

	required := 10
	oversampled := 3
	total := required + oversampled

	enc, err := reedsolomon.New(required, oversampled)
	if err != nil {
		panic(err)
	}
	data := make([][]byte, total)

	msg := "hello, brave new world!"
	data_bytes := []byte(msg)
	data_bytes = augmentByteArrayWithLength(data_bytes)
	data_bytes = pad(data_bytes, required)
	if len(data_bytes) % required != 0 {
		panic("Error: impoper padding!")
	}

	shard_size := len(data_bytes) / required 	
	shards := split(data_bytes, shard_size)

	// Create all shards, size them at 50000 each
    for i:=0; i< len(data); i++ {
		data_slice := make([]byte, shard_size)
		data[i] = data_slice
	  }
	// fmt.Println("shards: ", shards)
	  
	  
	// Fill some data into the data shards
	for i:=0; i < required; i++ {
		for j:=0; j < shard_size; j++ {
			data[i][j] = shards[i][j]
		} 
	}

	err = enc.Encode(data)
	if err != nil {
		panic(err)
	}

	ok, err := enc.Verify(data)
	fmt.Println("Verifying that last shards contain parity data:", ok)

	// Delete one data shard
    data[1] = nil
    
    // Reconstruct the missing shards
    err = enc.Reconstruct(data)
	if err != nil {
		panic(err)
	}

	padded_bytes := bytes.Join(data[:required], []byte(""))
	// fmt.Printf("Original Data: %s \n", original_bytes)
	// fmt.Println("Padded Data:", string(padded_bytes))

	original_bytes := getBytesFromAugmented(padded_bytes)
	original_msg := string(original_bytes)
	fmt.Println("Original data bytes:", original_msg)

	check := original_msg == msg
	fmt.Println("Verifying that original message matches the decoded message:", check)

}



// func main() {

// 	required := 8
// 	oversampled := 6
// 	total := required + oversampled

// 	// Create a *FEC, which will require required pieces for reconstruction at
// 	// minimum, and generate total total pieces.
// 	f, err := infectious.NewFEC(required, total)
// 	if err != nil {
// 		panic(err)
// 	}

// 	// Prepare to receive the shares of encoded data.
// 	shares := make([]infectious.Share, total)
// 	output := func(s infectious.Share) {
// 		// the memory in s gets reused, so we need to make a deep copy
// 		shares[s.Number] = s.DeepCopy()
// 	}

// 	msg := "hello, brave new world!"
// 	data_bytes := []byte(msg)
// 	data_bytes = augmentByteArrayWithLength(data_bytes)
// 	data_bytes = pad(data_bytes, required)
// 	if len(data_bytes) % required != 0 {
// 		panic("Error: impoper padding!")
// 	}

// 	// shard_size := len(data_bytes) / required 	
// 	// shards := split(data_bytes, shard_size)

// 	// data encoded
// 	err = f.Encode(data_bytes, output)
// 	if err != nil {
// 		panic(err)
// 	}

// 	// we now have total shares.
// 	for _, share := range shares {
// 		fmt.Printf("%d: %#v\n", share.Number, string(share.Data))
// 	}

// 	// Let's reconstitute with two pieces missing and one piece corrupted.
// 	shares = shares[2:]     // drop the first two pieces
// 	shares[2].Data[1] = '!' // mutate some data

// 	result, err := f.Decode(nil, shares)
// 	if err != nil {
// 		panic(err)
// 	}

// 	// we have the original data!
// 	fmt.Printf("got: %#v\n", string(result))


// 	padded_bytes := result
// 	// fmt.Printf("Original Data: %s \n", original_bytes)
// 	// fmt.Println("Padded Data:", string(padded_bytes))

// 	original_bytes := getBytesFromAugmented(padded_bytes)
// 	original_msg := string(original_bytes)
// 	fmt.Println("Original data bytes:", original_msg)

// 	check := original_msg == msg
// 	fmt.Println("Verifying that original message matches the decoded message:", check)


// }
