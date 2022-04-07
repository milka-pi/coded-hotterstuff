package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"sync"

	// use it as "hotstuff", e.g., hotstuff.Node{}
	"github.com/dshulyak/go-hotstuff/types"
)

// ASK: correct handling of cases? maybe need something other than default?
// DONE: Delegate waitgroup to parent method

// inMsgsChan: channel for incoming messages, from all nodes
func listenForMessages(ctx context.Context, idx int, conn net.Conn, inMsgsChan chan <- *types.Message) error {

	fmt.Println("Node ", idx, "--> ", "Listening for messages... ")
	reader := bufio.NewReader(conn)
	for {
		select {
			case <-ctx.Done():  // if cancelFunction() executes
				fmt.Println("listenForMessages: Time to return")
				return nil
			default:
				msg_bytes, readErr := getMessageFromReader(reader)
				// error handling
				if readErr != nil {
					// conn is probably closed
					fmt.Println("Error: failed to read data!")
					return readErr
				}
				fmt.Println("Node ", idx, "--> ", "Received message from buffer: ", msg_bytes) 
				msg := &types.Message{}
				msg.Unmarshal(msg_bytes)
				inMsgsChan <- msg
		}	
	}

}	


/*  Arguments:
		ctx: to support cancelFunction feature from parent method (handleConnection)
		conn: connection wiht specific node
		msgsChan: receiving-only channel which activates this method. Any message read from msgsChan will be sent across the conn
			TODO: should it be array of *types.Message ?
	Return:
		nil when cancelFunction is called from handleConnection
		error when conn.Write() returns an error
*/
//  DONE: Delegate waitgroup to parent method
func sendMessages(ctx context.Context, idx int, conn net.Conn, outMsgsChan <-chan *types.Message) error {
	for {
		select{
			case <- ctx.Done(): // if cancelFunction() executes
			fmt.Println("sendMessages: Time to return")
				return nil
				
			case msg := <-outMsgsChan:
				fmt.Println("Read msg from outMsgsChan...")
				bytes, _ := msg.Marshal()
				augBytes := augmentByteArrayWithLength(bytes)
				fmt.Println("Node ", idx, "--> ", "Sender --> message byteArray: ", augBytes)
				_, sendErr := conn.Write(augBytes)
				// error handling
				if sendErr != nil {
					// conn is probably closed
					fmt.Println("Error: failed to send data!")
					return sendErr
				}
		}
	}
}

func exchangeIDs(conn net.Conn, myID int) (int, error) {
	// send myID
	initMsg := toByteArray(strconv.Itoa(myID))
	_, sendErr := conn.Write(initMsg)
	if sendErr != nil {
		fmt.Println("initExchange: Failed to send my ID!")
		return -1, sendErr
	}

	// receive yourID
	reader := bufio.NewReader(conn) // Q: what happens when reader is discarded?
	msg_bytes, readErr := getMessageFromReader(reader) // will only read first message
	if readErr != nil {
		// conn is probably closed
		fmt.Println("initExchange: Failed to read your ID!")
		return -1, readErr
	}
	yourID, convertError := strconv.Atoi(toString(msg_bytes))
	if convertError != nil {
		fmt.Println("initExchange: Failed to convert string ID to int!")
		return -1, convertError
	}
	fmt.Println("Node ", myID, "--> ", "Received your ID from buffer: ", yourID)

	return yourID, nil
}

// both listening for and sending messages/requests 
func handleConnection(conn net.Conn, idx int, arrayOfChannels []chan *types.Message, inMsgsChan chan *types.Message) {
	
	//Make a background context
	ctx := context.Background()
	//Derive a context with cancel
	ctxWithCancel, cancelFunction := context.WithCancel(ctx)
	defer cancelFunction()

	// Init protocol: should be blocking
	myID := idx
	yourID, initError := exchangeIDs(conn, myID)
	if initError != nil {
		fmt.Println("Node ", idx, "--> ", "handleConnection: initError, closing connection and returning")
		conn.Close()
		return 
	}
	fmt.Println("Node ", idx, "--> ", "handleConnection: yourID = ", yourID)

	// index into array of channels at index yourID
	outMsgsChan := arrayOfChannels[yourID]

	// set up wait group: to know when to close the connection
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		listenMsgsErr := listenForMessages(ctxWithCancel, idx, conn, inMsgsChan) // returns if there is an error or ctx was cancelled
		if listenMsgsErr != nil {
			fmt.Println("handleConnection: listenForMessages returned error, calling cancelFunction")
			fmt.Println(listenMsgsErr)
		}
		cancelFunction() // should it be inside if statement?
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		sendMsgsErr := sendMessages(ctxWithCancel, idx, conn, outMsgsChan) // only returns if there is an error or ctx was cancelled
		if sendMsgsErr != nil {
			fmt.Println("handleConnection: sendMessages returned error, calling cancelFunction")
			fmt.Println(sendMsgsErr)
		}
		cancelFunction()
	}()

	wg.Wait()
	conn.Close()
}

// can accept multiple connections, blocks on each iteration at ln.Accept()
func acceptConnections(ln net.Listener, idx int, arrayOfChannels []chan *types.Message, inMsgsChan chan *types.Message) {
	for {
		conn, err_conn := ln.Accept()
		// error handling
		if err_conn != nil {
			fmt.Println("Error accepting: ", err_conn.Error())
		}
		fmt.Println("Established connection")

		go handleConnection(conn, idx, arrayOfChannels, inMsgsChan)
	}
}


func listenForConnections(address string, idx int, arrayOfChannels []chan *types.Message, inMsgsChan chan *types.Message) {
	ln, err := net.Listen(NETWORK_TYPE, address)
	// error handling
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Listening for requests...")

	go acceptConnections(ln, idx, arrayOfChannels, inMsgsChan);

}

func initiateConnection(address string, idx int, arrayOfChannels []chan *types.Message, inMsgsChan chan *types.Message) {

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
		go handleConnection(conn, idx, arrayOfChannels, inMsgsChan)
		wg.Wait()
	}

}


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