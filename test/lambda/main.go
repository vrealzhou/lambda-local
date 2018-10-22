package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
)

var returnMsg = flag.String("m", "Hello", "return message")

func handler(input json.RawMessage) (json.RawMessage, error) {
	fmt.Printf("Input: %s\n", string(input))
	return []byte(fmt.Sprintf(`{"Message": "%s"}`, *returnMsg)), nil
}

func main() {
	flag.Parse()
	os.Setenv("_LAMBDA_SERVER_PORT", "3001")
	lambda.Start(handler)
}
