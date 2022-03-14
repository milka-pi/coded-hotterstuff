package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
)


func listenForMessages(conn net.Conn) {
	for {
		fmt.Println("listening for messages... ")
		reader := bufio.NewReader(conn)
		for {
			msg := getMessageFromReader(reader)
			fmt.Println("Received message from buffer: ", msg)

			// fmt.Println("Echoing client message: ", netData)

			// BAD IDEA: keeps sending messages back and forth!
			// echoMessage(conn, netData)
		}
	}

}

// both listening for and sending messages/requests 
func handleConnection(conn net.Conn, sendMsg bool, msg string) {
	
	// QUESTION: how to include this line of code?
	// defer conn.Close()

	go listenForMessages(conn)

	if (sendMsg == true){
		go sendMessages(conn, msg)
	}
}

// currently functional to send <= 1 message.
// TODO: add buffer to allow for more messages
func sendMessages(conn net.Conn, msg string) {	
	fmt.Println("Sending message...")
	b := toByteArray(msg)

	// ISSUE: send same message twice --> cannot properly receive both messages from buffer.
	// fixed ? two for loops in listenForMessages
	b2 := toByteArray(msg)
	b = append(b, b2...)

	fmt.Println("Sender --> message byteArray: ", b)

	_, sendErr := conn.Write(b)
	// error handling
	if sendErr != nil {
		fmt.Println("Failed to send data!")
	}

}


// not used currently
func echoMessage(conn net.Conn, msg string) {
	server_reply := fmt.Sprintf("Server reply: %s", msg)
	b := toByteArray(server_reply)
	// error handling
	_, err := conn.Write(b)
	if err != nil {
		fmt.Println("Server --> Failed to write!")
	}
}

// can accept multiple connections?
func acceptConnections(ln net.Listener, sendMsg bool, msg string) {
	for {
		conn, err_conn := ln.Accept()
		// error handling
		if err_conn != nil {
			fmt.Println("Error accepting: ", err_conn.Error())
		}
		fmt.Println("Established connection")

		go handleConnection(conn, sendMsg, msg)
	}
}

// problem: if listening, cannot listen from > 1 node
// DONE: fix this --> FIXED with 'acceptConnections' goroutine
func tcpNodeFunc(mode string, address string, sendMsg bool, msg string) {
	// var conn net.Conn // common for both listener and initiator
	// var err_conn error

	if mode == "listen" {

		ln, err := net.Listen(NETWORK_TYPE, address)
		// error handling
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Listening for requests...")

		go acceptConnections(ln, sendMsg, msg);
	
	} else if mode == "initiate" {
		conn, err_conn := net.Dial(NETWORK_TYPE, address)
		// error handling
		if err_conn != nil {
			fmt.Println("Error accepting: ", err_conn.Error())
			return
		}
		fmt.Println("Established connection")

		go handleConnection(conn, sendMsg, msg)
	}
	
	// go handleConnection(conn, sendMsg, msg)

}