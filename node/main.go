package main

import (
	"flag"
	"fmt"
)

const (
	NETWORK_TYPE = "tcp"
	DEFAULT_ADDRESS = ":8080"
	DEFAULT_MESSAGE = "hello"
)

func main() {

	// nodeFlag can take two values: "server" or "client"
	var nodeFlag string
	var addrFlag string
	var msgFlag string
	flag.StringVar(&nodeFlag, "node", "server", "server or client node")
	flag.StringVar(&addrFlag, "addr", DEFAULT_ADDRESS, "port address")
	flag.StringVar(&msgFlag, "msg", DEFAULT_MESSAGE, "message to send to the server")

	flag.Parse()

	if nodeFlag == "server" {
		fmt.Println("Running server code!")
		tcpServerFunc(addrFlag)

	} else if nodeFlag == "client" {
		fmt.Println("Running client code!")
		tcpClientFunc(addrFlag, msgFlag)

	} else {
		panic("nodeFlag argument only accepts values 'server' and 'client'");
	}
	
}
