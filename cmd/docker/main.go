package main

import (
	"fmt"
	"log"
	"os"

	"github.com/namsral/flag"
	"github.com/vrealzhou/lambda-local/internal/invoker"
	"github.com/vrealzhou/lambda-local/internal/template"
	"gopkg.in/yaml.v2"
)

type arguments struct {
	port     int
	template string
}

var functions map[string]template.Function

func main() {
	args := parseArgs()
	flag.Parse()
	invoker.LambdaBinBase = "build/lambdas"
	tmpl := parseTemplate(args.template)
	functions = tmpl.Functions()
	funcName := "Test"
	defer clearUp()
	payload := []byte(`{
		"action":"create",
		"contenttype":"release",
		"contentid":"mrKyQomBA9",
		"contentversion":1,
		"contentsource":"mapi1"
	}`)
	result, err := invoke(funcName, payload)
	if err != nil {
		log.Fatal("lambda error:", err)
	}
	fmt.Printf("lambda: %s\n", string(result))
}

func invoke(name string, payload []byte) ([]byte, error) {
	err := invoker.PrepareFunction(name, functions[name])
	if err != nil {
		return nil, fmt.Errorf("Error on prepar function Test: %s", err.Error())
	}
	meta := invoker.Functions["Test"]
	return invoker.InvokeFunc(meta, payload)
}

func clearUp() {
	for name, f := range invoker.Functions {
		log.Printf("Stop function %s, process id: %d", name, f.Pid)
		proc, err := os.FindProcess(f.Pid)
		if err != nil {
			log.Printf("Error on find process: %d", f.Pid)
		}
		proc.Kill()
	}
}

func parseTemplate(tmplFile string) template.SAMTemplate {
	if tmplFile == "" {
		panic("Please specify template file via -template={filename}")
	}
	f, err := os.Open(tmplFile)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	d := yaml.NewDecoder(f)
	d.SetStrict(false)
	tmpl := template.SAMTemplate{}
	err = d.Decode(&tmpl)
	if err != nil {
		panic(err)
	}
	return tmpl
}

func parseArgs() arguments {
	var args arguments
	flag.IntVar(&args.port, "port", 3001, "server port")
	flag.StringVar(&args.template, "template", "deployments/ingestor-sam.yaml", "SAM template file")
	flag.Parse()
	return args
}
