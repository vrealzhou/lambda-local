package main

import (
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/vrealzhou/lambda-local/test"
)

type Message struct {
	Message string `json:"message,omitempty"`
}

func handler(input test.Input) (Message, error) {
	fmt.Printf("Input: %v\n", input)
	msg := Message{
		Message: fmt.Sprintf("Cheers %s!", input.Name),
	}
	for _, env := range os.Environ() {
		fmt.Println("\t", env)
	}
	return msg, nil
}

func main() {
	lambda.Start(handler)
}
