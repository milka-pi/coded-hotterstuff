package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
)


func handleConnection(conn net.Conn) {
	// defer conn.Close()

	// fmt.Println("Received request")
	netData, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Received message from buffer: ", netData)
	// fmt.Println("Echoing client message: ", netData)
	// echoMessage(c, netData)

}

func sendMessage(conn net.Conn, msg string) {

	fmt.Println("Sending message...")
	b := []byte(msg + "\n")
	_, sendErr := conn.Write(b)
	if sendErr != nil {
		fmt.Println("Failed to send data!")
	}

}

func echoMessage(c net.Conn, msg string) {
	_, err := c.Write([]byte(fmt.Sprintf("server replied: %s\n", msg)))
	if err != nil {
		fmt.Println("server: failed to write!")
	}
}

func tcpNodeFunc(mode string, address string, sendMsg bool, msg string) {
	var conn net.Conn // common for both listener and initiator
	var err_conn error
	if mode == "listen" {
		ln, err := net.Listen(NETWORK_TYPE, address)
		fmt.Println("Listening for requests...")
		if err != nil {
			log.Fatal(err)
		}

		conn, err_conn = ln.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err_conn.Error())
		}
		fmt.Println("Established connection")
	
	} else if mode == "initiate" {
		conn, err_conn = net.Dial(NETWORK_TYPE, address)
		if err_conn != nil {
			fmt.Println("Error accepting: ", err_conn.Error())
			return
		}
		fmt.Println("Established connection")
	}

	
	sent := false

	for {
		// go handleConnection(conn)
		go handleConnection(conn)
		
		// fmt.Println("bool sendMsg:", sendMsg)
		if (sendMsg && sent == false) {
			sendMessage(conn, msg)
			sent = true
		}


	}

}