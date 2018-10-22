package main

import (
	"fmt"
	"log"
	"os"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/namsral/flag"
	"github.com/vrealzhou/lambda-local/internal/invoker"
	"github.com/vrealzhou/lambda-local/internal/template"
	"gopkg.in/yaml.v2"
	"github.com/labstack/echo"
)

type arguments struct {
	port     int
	template string
}

var functions map[string]template.Function

func main() {
	args := parseArgs()
	flag.Parse()
	tmpl := parseTemplate(args.template)
	functions = tmpl.Functions()
	defer clearUp()
	serve(args)
}

func serve(args arguments) {
	e := echo.New()
	e.POST("/2015-03-31/functions/:function/invocations", invoke)
	e.Logger.Fatal(e.Start(":"+strconv.Itoa(args.port)))
}

func invoke(c echo.Context) error {
	name := c.Param("function")
	err := invoker.PrepareFunction(name, functions[name])
	if err != nil {
		return fmt.Errorf("Error on prepar function Test: %s", err.Error())
	}
	meta := invoker.Functions["Test"]
	payload, err := ioutil.ReadAll(c.Request().Body)
	defer c.Request().Body.Close()
	if err != nil {
		return err
	}
	result, err := invoker.InvokeFunc(meta, payload)
	if err != nil {
		return err
	}
	c.JSON(http.StatusOK, result)
	return nil
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
