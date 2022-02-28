package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
)


func handleConnection(c net.Conn) {
	defer c.Close()

	fmt.Println("Received request")
	netData, err := bufio.NewReader(c).ReadString('\n')
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Echoing client message: ", netData)
	echoMessage(c, netData)
}

func echoMessage(c net.Conn, msg string) {
	_, err := c.Write([]byte(fmt.Sprintf("server replied: %s\n", msg)))
	if err != nil {
		fmt.Println("server: failed to write!")
	}
}

func tcpServerFunc(address string) {
	ln, err := net.Listen(NETWORK_TYPE, address)
	fmt.Println("Listening for requests...")
	if err != nil {
		log.Fatal(err)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
            continue
		}
		go handleConnection(conn)
	}
}