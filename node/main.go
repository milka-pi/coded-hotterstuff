package main

import (
	"flag"
	"fmt"
	"strconv"
	// "github.com/dshulyak/go-hotstuff"	// use it has "hotstuff", e.g., hotstuff.Node{}
	// "fmt"
)

const (
	NETWORK_TYPE = "tcp"
	DEFAULT_ADDRESS_NUMBER = 8000
	DEFAULT_MESSAGE = "hello"
	NUMBER_OF_NODES = 4
)

func getAddressList() [NUMBER_OF_NODES]string {
	addressList := [NUMBER_OF_NODES]string{}
	for i := 0; i < NUMBER_OF_NODES; i++ {
		addressList[i] = ":" + strconv.Itoa(DEFAULT_ADDRESS_NUMBER + i)
	}
	return addressList
}

func checkFlags(mode string) {
	if mode != "listen" && mode != "initiate" {
		panic("nodeMode argument only accepts values 'listen' and 'initiate'");
	}
	if mode != "listen" && mode != "initiate" {
		panic("nodeMode argument only accepts values 'listen' and 'initiate'");
	}
}

func main() {

	// mode can take two values: "listen" or "initiate"
	var index int
	var mode string //not used
	var address string //not used
	var sendMsg bool
	var msg string
	flag.IntVar(&index, "index", 0, "node index")
	flag.StringVar(&mode, "mode", "listen", "listen for connection or initiate connection (initially)") //not used
	flag.StringVar(&address, "addr", ":8080", "port address") // not used
	flag.BoolVar(&sendMsg, "sendMsg", false, "send message or not")
	flag.StringVar(&msg, "msg", DEFAULT_MESSAGE, "message to send. Ignored if sendMsg = False")

	flag.Parse()
	checkFlags(mode)

	fmt.Println("Node index:", index)
	fmt.Println("bool sendMsg:", sendMsg)


	addressList := getAddressList()


	msg = "hello from node " + strconv.Itoa(index)
	nodeAddress := addressList[index]
	go listenForConnections(nodeAddress, sendMsg, msg)

	for i := 0; i < index; i++ {
		fmt.Println("initiating connection to node with index:", i)
		address_i := addressList[i]
		go initiateConnection(address_i, sendMsg, msg)
	}

	// go tcpNodeFunc(mode, address, sendMsg, msg)
	select{}
}


