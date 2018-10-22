package invoke

import (
	"fmt"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
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
	req := &lambda.InvokeInput{
		FunctionName: aws.String("Test"),
		Payload: []byte(`{
			"action":"create",
			"contenttype":"release",
			"contentid":"mrKyQomBA9",
			"contentversion":1,
			"contentsource":"mapi1"
		}`),
	}
	resp, err := client.Invoke(req)
	if err != nil {
		b.Error(err)
	}
	fmt.Printf("Response: %s\n", string(resp.Payload))
}
