package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
)

var returnMsg = "Hello"

type Message struct {
	Message string `json:"message,omitempty"`
}

func handler(input json.RawMessage) (Message, error) {
	fmt.Printf("Input: %s\n", string(input))
	msg := Message{
		Message: returnMsg,
	}
	return msg, nil
}

func main() {
	returnMsg = os.Getenv("MESSAGE")
	fmt.Printf("Port: %s, Message: %s\n", os.Getenv("_LAMBDA_SERVER_PORT"), returnMsg)
	lambda.Start(handler)
}
