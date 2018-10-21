package main

import (
	"fmt"
	"log"
	"net"
	"net/rpc"

	"github.com/aws/aws-lambda-go/lambda/messages"
	"github.com/vrealzhou/lambda-local/test/server"
)

func main() {
	client, err := rpc.Dial("tcp", "localhost:3001")
	if err != nil {
		log.Fatal("dial error:", err)
	}
	defer client.Close()
	req := &messages.InvokeRequest{
		Payload:   []byte("{}"),
		RequestId: "12345",
		Deadline: messages.InvokeRequest_Timestamp{
			Seconds: 300,
		},
		InvokedFunctionArn: "Test",
	}
	response := &messages.InvokeResponse{}
	err = client.Call("Function.Invoke", req, response)
	if err != nil {
		log.Fatal("lambda error:", err)
	}
	fmt.Printf("lambda: %s\n", string(response.Payload))
	// StartServer()
	// var (
	// 	addr     = "127.0.0.1:3001"
	// 	request  = &server.Request{Name: "Request"}
	// 	response = new(server.Response)
	// )
	// client, err := rpc.Dial("tcp", addr)
	// if err != nil {
	// 	log.Fatal("dial error:", err)
	// }
	// // defer client.Close()
	// err = client.Call(server.HandlerName, request, response)
	// if err != nil {
	// 	log.Fatal("arith error:", err)
	// }
	// fmt.Println(response.Message)
}

func StartServer() {
	go func() {
		rpc.Register(&server.Handler{})

		// Create a TCP listener that will listen on `Port`
		listener, err := net.Listen("tcp", ":3001")
		if err != nil {
			log.Fatal("listen error:", err)
		}

		// Close the listener whenever we stop
		//defer listener.Close()

		// Wait for incoming connections
		rpc.Accept(listener)
	}()
}
