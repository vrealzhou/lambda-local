package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	"github.com/awslabs/goformation/cloudformation/resources"
	"github.com/labstack/echo"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	config "github.com/vrealzhou/lambda-local/config/docker"
	"github.com/vrealzhou/lambda-local/internal/invoker"
	"github.com/vrealzhou/lambda-local/internal/template"
)

var functions map[string]*resources.AWSServerlessFunction

var rootCmd = &cobra.Command{
	Use:   "run",
	Short: "Run lambda service",
	Long:  "Run lambda service",
	Run: func(cmd *cobra.Command, args []string) {
		envFile := config.EnvFile()
		if envFile != "" {
			err := invoker.LoadEnvFile(envFile)
			if err != nil {
				log.Fatal(err)
			}
		}
		var err error
		functions, err = parseTemplate(config.Template())
		if err != nil {
			log.Fatal(err)
		}
		defer clearUp()
		serve()
	},
}

func main() {
	config.ParseArgs(rootCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func serve() {
	e := echo.New()
	e.POST("/2015-03-31/functions/:function/invocations", invoke)
	e.Logger.Fatal(e.Start(":" + strconv.Itoa(config.Port())))
}

func invoke(c echo.Context) error {
	name := c.Param("function")
	err := invoker.PrepareFunction(name, functions[name])
	if err != nil {
		return fmt.Errorf("Error on prepar function Test: %s", err.Error())
	}
	h, ok := invoker.Functions[name]
	if !ok {
		return fmt.Errorf("Unregistered function name: %s", name)
	}
	payload, err := ioutil.ReadAll(c.Request().Body)
	defer c.Request().Body.Close()
	if err != nil {
		return err
	}
	result, err := h.Invoke(payload)
	if err != nil {
		if result != nil {
			c.Response().Header().Set("X-Amz-Executed-Version", "$LATEST")
			c.Response().Header().Set("X-Amz-Function-Error", "Unhandled")
			c.JSONBlob(http.StatusOK, result)
			return nil
		}
		return err
	}
	c.JSON(http.StatusOK, result)
	return nil
}

func clearUp() {
	for name, f := range invoker.Functions {
		log.Debugf("Stop function %s", name)
		err := f.Stop()
		if err != nil {
			log.Errorf("Error on stop function: %s", name)
		}
	}
}

func parseTemplate(tmplFile string) (map[string]*resources.AWSServerlessFunction, error) {
	if tmplFile == "" {
		return nil, fmt.Errorf("Please specify template file via --template {filename}")
	}
	return template.Parse(tmplFile, nil)
}
