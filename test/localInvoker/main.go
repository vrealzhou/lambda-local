package main

import (
	"fmt"
	"log"
	"net/rpc"

	"github.com/aws/aws-lambda-go/lambda/messages"
)

func main() {
	client, err := rpc.Dial("tcp", "localhost:3001")
	if err != nil {
		log.Fatal("dial error:", err)
	}
	defer client.Close()
	req := &messages.InvokeRequest{
		Payload: []byte(`{
			"action":"create",
			"contenttype":"release",
			"contentid":"mrKyQomBA9",
			"contentversion":1,
			"contentsource":"mapi"
		}`),
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
}
