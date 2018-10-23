package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/vrealzhou/lambda-local/test"
)

var returnMsg = "Hello"

type Message struct {
	Message string `json:"message,omitempty"`
}

func handler(ctx context.Context, input test.Input) (Message, error) {
	fmt.Printf("Input: %v\n", input)
	deadline, ok := ctx.Deadline()
	if ok {
		fmt.Printf("Deadline: %s\n", deadline.Format("2006-01-02 15:04:05"))
	}
	msg := Message{
		Message: fmt.Sprintf("%s %s", returnMsg, input.Name),
	}
	return msg, nil
}

func main() {
	returnMsg = os.Getenv("MESSAGE")
	lambda.Start(handler)
}
