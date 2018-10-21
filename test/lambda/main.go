package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
)

func handler(input json.RawMessage) (json.RawMessage, error) {
	fmt.Printf("Input: %s\n", string(input))
	return []byte(`{"Result": "Hello"}`), nil
}

func main() {
	os.Setenv("_LAMBDA_SERVER_PORT", "3001")
	lambda.Start(handler)
}
