package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"sync"
	"time"

	// use it as "hotstuff", e.g., hotstuff.Node{}
	"github.com/dshulyak/go-hotstuff/types"
	"github.com/xtaci/kcp-go/v5"	
)

// ASK: correct handling of cases? maybe need something other than default?
// DONE: Delegate waitgroup to parent method

// inMsgsChan: channel for incoming messages, from all nodes
func listenForMessages(ctx context.Context, idx int, conn io.Reader, inMsgsChan chan <- *types.Message) error {

	fmt.Println("Node ", idx, "--> ", "Listening for messages... ")
	for {
		select {
			case <-ctx.Done():  // if cancelFunction() executes
				fmt.Println("listenForMessages: Time to return")
				return nil
			default:
				msg_bytes, readErr := getMessageFromReader(conn)
				// error handling
				if readErr != nil {
					// conn is probably closed
					fmt.Println("Error: failed to read data!")
					return readErr
				}
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
				// fmt.Println("Read msg from outMsgsChan...")
				// fmt.Println("node index: ", idx, "Message size: ", msg.Size())
				bytes, errMarshal := msg.Marshal()
				if errMarshal != nil {
					fmt.Println("Error attempting to marshall message: ", msg)
					// outMsgsChan <-msg
					// return errMarshal
				}
				augBytes := augmentByteArrayWithLength(bytes)
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

func exchangeIDs(conn io.ReadWriter, myID int) (int, error) {
	// send myID
	initMsg := toByteArray(strconv.Itoa(myID))
	_, sendErr := conn.Write(initMsg)
	if sendErr != nil {
		fmt.Println("initExchange: Failed to send my ID!")
		return -1, sendErr
	}

	// receive yourID
	msg_bytes, readErr := getMessageFromReader(conn) // will only read first message
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

	return yourID, nil
}

// both listening for and sending messages/requests 
func handleConnection(conn net.Conn, idx int, arrayOfChannels []chan *types.Message, inMsgsChan chan *types.Message) {
	// fmt.Printf("node %v has new connection\n", idx)

	//Make a background context
	ctx := context.Background()
	//Derive a context with cancel
	ctxWithCancel, cancelFunction := context.WithCancel(ctx)
	defer cancelFunction()

	// Init protocol: should be blocking
	myID := idx
	yourID, initError := exchangeIDs(conn, myID)
	if initError != nil {
		fmt.Println("error exchanging ID:", initError)
		conn.Close()
		return
	}
	fmt.Printf("connection established %v - %v\n", idx, yourID)

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
		// fmt.Println("Established connection")

		go handleConnection(conn, idx, arrayOfChannels, inMsgsChan)
	}
}


func listenForConnections(ipAddress string, idx int, arrayOfChannels []chan *types.Message, inMsgsChan chan *types.Message) {
	block, _ := kcp.NewNoneBlockCrypt(nil)
	ln, err := kcp.ListenWithOptions(ipAddress, block, 10, 3);
	// error handling
	if err != nil {
		log.Fatal(err)
	}
	// fmt.Println("Listening for requests...")

	go acceptConnections(ln, idx, arrayOfChannels, inMsgsChan);

}

func initiateConnection(ipAddress string, idx int, arrayOfChannels []chan *types.Message, inMsgsChan chan *types.Message) {
	fmt.Printf("node %v initiating connection to %v\n", idx, ipAddress)
	// if connection is closed, try to Dial again.
	for {
		block, _ := kcp.NewNoneBlockCrypt(nil)
		conn, err_conn := kcp.DialWithOptions(ipAddress, block, 10, 3)
		// error handling
		if err_conn != nil {
			fmt.Printf("node %v initiating connection to %v: %v\n", idx, ipAddress, err_conn)
			time.Sleep(time.Duration(100) * time.Millisecond)
		} else {
			fmt.Printf("node %v successfully initiated connection to %v\n", idx, ipAddress)

			var wg sync.WaitGroup
			wg.Add(1)
			go handleConnection(conn, idx, arrayOfChannels, inMsgsChan)
			wg.Wait()
		}
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
