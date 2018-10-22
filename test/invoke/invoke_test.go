package invoke

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/vrealzhou/lambda-local/test"
)

var client *lambda.Lambda

func setup() {
	sess := session.New()
	client = lambda.New(sess, &aws.Config{
		MaxRetries: aws.Int(3),
		Endpoint:   aws.String("http://localhost:3001"),
		Region:     aws.String("ap-southeast-2"),
	})
}

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	os.Exit(code)
}

func BenchmarkLambdaInvoke(b *testing.B) {
	input := test.Input{
		Name: "Bruce Zhou",
	}
	payload, _ := json.Marshal(input)
	req := &lambda.InvokeInput{
		FunctionName: aws.String("Hello"),
		Payload:      payload,
	}
	resp, err := client.Invoke(req)
	if err != nil {
		b.Error(err)
	}
	fmt.Printf("Response: %s\n", string(resp.Payload))
}
