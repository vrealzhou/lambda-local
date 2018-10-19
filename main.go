package main

import (
	"fmt"
	"os"

	"github.com/namsral/flag"
	"gopkg.in/yaml.v2"
)

type arguments struct {
	port     int
	profile  string
	template string
}

func main() {
	args := parseArgs()
	if args.template == "" {
		fmt.Printf("argument -template must be set")
		return
	}
	template := parseTemplate(args)
	for name, f := range template.Functions() {
		fmt.Printf("Function %s: %#v\n", name, f)
	}
}

func parseTemplate(args arguments) SAMTemplate {
	f, err := os.Open(args.template)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	d := yaml.NewDecoder(f)
	d.SetStrict(false)
	template := SAMTemplate{}
	err = d.Decode(&template)
	if err != nil {
		panic(err)
	}
	return template
}

func parseArgs() arguments {
	var args arguments
	flag.IntVar(&args.port, "port", 3001, "server port")
	flag.StringVar(&args.profile, "profile", "default", "AWS profile")
	flag.StringVar(&args.template, "template", "", "SAM template file")
	flag.Parse()
	return args
}
