package main

import (
	"flag"
	"fmt"
	// "github.com/dshulyak/go-hotstuff"	// use it has "hotstuff", e.g., hotstuff.Node{}
	// "fmt"
)

const (
	NETWORK_TYPE = "tcp"
	DEFAULT_ADDRESS = ":8080"
	DEFAULT_MESSAGE = "hello"
)


// func checkFlags(mode string) {
// 	if mode != "listen" && mode != "initiate" {
// 		panic("nodeMode argument only accepts values 'listen' and 'initiate'");
// 	}
// 	if mode != "listen" && mode != "initiate" {
// 		panic("nodeMode argument only accepts values 'listen' and 'initiate'");
// 	}
// }

func main() {

	// mode can take two values: "listen" or "initiate"
	var mode string
	var address string
	var sendMsg bool
	var msg string
	flag.StringVar(&mode, "mode", "listen", "listen for connection or initiate connection (initially)")
	flag.StringVar(&address, "addr", DEFAULT_ADDRESS, "port address")
	flag.BoolVar(&sendMsg, "sendMsg", false, "send message or not")
	flag.StringVar(&msg, "msg", DEFAULT_MESSAGE, "message to send. Ignored if sendMsg = False")

	flag.Parse()

	// checkFlags(mode)

	fmt.Println("bool sendMsg:", sendMsg)
	go tcpNodeFunc(mode, address, sendMsg, msg)
	select{}
}


