package main

// import (
// 	"bufio"
// 	"fmt"
// 	"net"
// )

// func tcpClientFunc(address string, message string) {
// 	conn, err := net.Dial(NETWORK_TYPE, address)
// 	if err != nil {
// 		fmt.Println("Error accepting: ", err.Error())
//         return
// 	}
// 	// fmt.Fprintf(conn, "GET / HTTP/1.0\r\n\r\n")

// 	b := []byte(message + "\n" + "hi again" + "\n")
//     _, sendErr := conn.Write(b)
// 	if sendErr != nil {
// 		fmt.Println("Failed to send data to the server!")
// 	}

// 	status, readErr := bufio.NewReader(conn).ReadString('\n')
// 	if readErr != nil {
// 		fmt.Println("Failed to send data to the server!")
// 	}
// 	fmt.Println(status)
// }