package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"sync"
)


func listenForMessages(ctx context.Context, conn net.Conn, wg sync.WaitGroup) {

	defer wg.Done() // NEW: is it correct?

	// try deleting outer for loop
	// TODO: did it work?

	fmt.Println("listening for messages... ")
	reader := bufio.NewReader(conn)
	for {
		// wait for error too
		msg, readErr := getMessageFromReader(reader)
		if readErr != nil {
			// write on channel? do anything else?
			return
		}
		fmt.Println("Received message from buffer: ", msg)		
	}
}


// select {
// 	case <-ctx.Done():  // if cancel() execute
// 		fmt.Println("listenForMessages: Time to return")
// 		// return
// 	default:
// 		// wait for error too
// 		msg, readErr := getMessageFromReader(reader)
// 		if readErr != nil {
// 			// write on channel ?
// 			// do anything else ?
// 			return
// 		}
// 		fmt.Println("Received message from buffer: ", msg)               
// }	


// currently functional to send <= 1 message.
// TODO: add channel to allow for more messages
func sendMessages(ctx context.Context, conn net.Conn, msg string) {	
	fmt.Println("Sending message...")
	b := toByteArray(msg)

	// ISSUE: send same message twice --> cannot properly receive both messages from buffer.
	// fixed : keep reading from reader
	// b2 := toByteArray(msg)
	// b = append(b, b2...)

	fmt.Println("Sender --> message byteArray: ", b)


	// 

	_, sendErr := conn.Write(b)
	// error handling
	if sendErr != nil {
		fmt.Println("Failed to send data!")
	}

}

// both listening for and sending messages/requests 
func handleConnection(conn net.Conn, sendMsg bool, msg string) {
	
	// QUESTION: how to include this line of code? Answer: do not include as is!
	// defer conn.Close()
	
	//Make a background context
	ctx := context.Background()
	//Derive a context with cancel
	ctxWithCancel, cancelFunction := context.WithCancel(ctx)
	defer cancelFunction()

	// set up wait group: to know when to close the connection
	var wg sync.WaitGroup

	wg.Add(1)
	go listenForMessages(ctxWithCancel, conn, wg) // only returns if there is an error

	// 1st approach: set up channel, pass it as argument, and signal message when go routine has to terminate
	// 2nd approach: use context package with cancelFunction
	if (sendMsg == true){
		wg.Add(1)
		go sendMessages(ctxWithCancel, conn, msg)
	}

	wg.Wait()
	conn.Close()
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


func listenForConnections(address string, sendMsg bool, msg string) {
	ln, err := net.Listen(NETWORK_TYPE, address)
	// error handling
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Listening for requests...")

	go acceptConnections(ln, sendMsg, msg);

}

func initiateConnection(address string, sendMsg bool, msg string) {

	// if connection is closed, try to Dial again.
	for {
		conn, err_conn := net.Dial(NETWORK_TYPE, address)
		// error handling
		if err_conn != nil {
			fmt.Println("Error initiating connection: ", err_conn.Error())
			return
		}
		fmt.Println("Established connection")

		var wg sync.WaitGroup
		wg.Add(1)
		go handleConnection(conn, sendMsg, msg)
		wg.Wait()
	}

}


// problem: if listening, cannot listen from > 1 node
// FIXED with 'acceptConnections' goroutine
// func tcpNodeFunc(mode string, address string, sendMsg bool, msg string) {

// 	if mode == "listen" {

// 		ln, err := net.Listen(NETWORK_TYPE, address)
// 		// error handling
// 		if err != nil {
// 			log.Fatal(err)
// 		}
// 		fmt.Println("Listening for requests...")

// 		go acceptConnections(ln, sendMsg, msg);
	
// 	} else if mode == "initiate" {

// 		// for loop (list of addresses to connect to)
// 		// go initiateConnection()

// 		address_list := []string{address}
// 		for i := 0; i < len(address_list); i++ {
// 			address_i := address_list[i]
// 			go initiateConnection(address_i, sendMsg, msg)
// 		}

// 		// TODO: for loop wrapping this, add wait group, and net.Dial again to re-establish connection. 
// 	}

// }




//-----------------------------------------------------------------------------------------------------------------

// not currently used
func echoMessage(conn net.Conn, msg string) {
	server_reply := fmt.Sprintf("Server reply: %s", msg)
	b := toByteArray(server_reply)
	// error handling
	_, err := conn.Write(b)
	if err != nil {
		fmt.Println("Server --> Failed to write!")
	}
}