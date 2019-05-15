package template

import (
	"github.com/vrealzhou/goformation"
	"github.com/vrealzhou/goformation/cloudformation/resources"
	"github.com/vrealzhou/goformation/intrinsics"
)

func Parse(file string, parameters map[string]string) (map[string]*resources.AWSServerlessFunction, error) {
	paramOverrides := make(map[string]interface{})
	for k, v := range parameters {
		paramOverrides[k] = v
	}
	// Open a template from file (can be JSON or YAML)
	tmpl, err := goformation.OpenWithOptions(file, &intrinsics.ProcessorOptions{
		ParameterOverrides: paramOverrides,
	}, nil)
	if err != nil {
		panic(err)
	}

	// You can extract all resources of a certain type
	// Each AWS CloudFormation resource is a strongly typed struct
	functions := tmpl.GetAllAWSServerlessFunctionResources()
	return functions, nil
}
