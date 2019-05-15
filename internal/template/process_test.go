package template

import (
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	// Open a template from file (can be JSON or YAML)
	functions, err := Parse("/Users/zhouy4l/go/src/stash.abc-dev.net.au/ter/serverless-event-management/deployments/aws/ingestor-sam.yaml", map[string]string{
		"HostEnv":        "staging",
		"DeployTime":     time.Now().String(),
		"QueueStackName": "QueueStack",
		"CacheHost":      "redis:6379",
	})
	if err != nil {
		t.Fatalf("There was an error processing the template: %s", err)
	}
	for name, function := range functions {
		// E.g. Found a AWS::Serverless::Function named GetHelloWorld (runtime: nodejs6.10)
		t.Logf("Found a %s named %s (runtime: %s)\n", function.AWSCloudFormationType(), name, function.Runtime)
	}
}
