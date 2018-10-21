package invoker

import (
	"log"
	"net/rpc"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/lambda/messages"
	uuid "github.com/satori/go.uuid"
)

var Functions map[string]FunctionMeta

type FunctionMeta struct {
	Name       string
	Arn        string
	Port       int
	TimeoutSec int64
}

func InvokeFunc(meta FunctionMeta, payload []byte) ([]byte, error) {
	start := time.Now()
	log.Printf("Start Invoke Function %s at: %s\n", meta.Name, start.Format("2006/01/02 15:04:05"))
	client, err := rpc.Dial("tcp", "localhost:"+strconv.Itoa(meta.Port))
	if err != nil {
		log.Fatal("dial error:", err)
	}
	defer client.Close()
	u1 := uuid.NewV4()
	req := &messages.InvokeRequest{
		Payload:   payload,
		RequestId: u1.String(),
		Deadline: messages.InvokeRequest_Timestamp{
			Seconds: meta.TimeoutSec,
		},
		InvokedFunctionArn: meta.Arn,
	}
	response := &messages.InvokeResponse{}
	invokeStart := time.Now()
	err = client.Call("Function.Invoke", req, response)
	invokeEnd := time.Now()
	log.Printf("Invoke Function %s preparing took: %s; Invoke took: %s; Total time cost: %s\n", meta.Name, invokeStart.Sub(start), invokeEnd.Sub(invokeStart), invokeEnd.Sub(start))
	if err != nil {
		return nil, err
	}
	return response.Payload, nil
}
