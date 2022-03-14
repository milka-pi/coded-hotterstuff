package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
)


func listenMessages(conn net.Conn) {
	for {
		fmt.Println("listening for messages... ")
		netData, err := bufio.NewReader(conn).ReadString('\n')
		// error handling
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("Received message from buffer: ", netData)
		// fmt.Println("Echoing client message: ", netData)
		// echoMessage(c, netData)
	}

}

// both listening for and sending messages/requests 
func handleConnection(conn net.Conn, sendMsg bool, msg string) {
	// QUESTION: how to include this line of code?
	// defer conn.Close()

	go listenMessages(conn)

	if (sendMsg == true){
		go sendMessages(conn, msg)
	}
}

func sendMessages(conn net.Conn, msg string) {
	// currently functional to send <= 1 message.
	// TODO: add buffer to allow for more messages
	
	fmt.Println("Sending message...")
	b := []byte(msg + "\n")
	_, sendErr := conn.Write(b)
	// error handling
	if sendErr != nil {
		fmt.Println("Failed to send data!")
	}

}

func echoMessage(c net.Conn, msg string) {
	_, err := c.Write([]byte(fmt.Sprintf("server replied: %s\n", msg)))
	// error handling
	if err != nil {
		fmt.Println("server: failed to write!")
	}
}

func tcpNodeFunc(mode string, address string, sendMsg bool, msg string) {
	var conn net.Conn // common for both listener and initiator
	var err_conn error
	if mode == "listen" {
		ln, err := net.Listen(NETWORK_TYPE, address)
		// error handling
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Listening for requests...")
		conn, err_conn = ln.Accept()
		// error handling
		if err != nil {
			fmt.Println("Error accepting: ", err_conn.Error())
		}
		fmt.Println("Established connection")
	
	} else if mode == "initiate" {
		conn, err_conn = net.Dial(NETWORK_TYPE, address)
		// error handling
		if err_conn != nil {
			fmt.Println("Error accepting: ", err_conn.Error())
			return
		}
		fmt.Println("Established connection")
	}
	
	go handleConnection(conn, sendMsg, msg)

}