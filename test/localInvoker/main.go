package main

import (
	"fmt"
	"log"

	"github.com/vrealzhou/lambda-local/internal/invoker"
)

func main() {
	payload, err := invoker.InvokeFunc(invoker.FunctionMeta{
		Name:       "Test",
		Arn:        "Test",
		Port:       3001,
		TimeoutSec: 300,
	}, []byte(`{
		"action":"create",
		"contenttype":"release",
		"contentid":"mrKyQomBA9",
		"contentversion":1,
		"contentsource":"mapi"
	}`))
	if err != nil {
		log.Fatal("lambda error:", err)
	}
	fmt.Printf("lambda: %s\n", string(payload))
}
