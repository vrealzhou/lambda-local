package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/labstack/echo"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	config "github.com/vrealzhou/lambda-local/config/docker"
	"github.com/vrealzhou/lambda-local/internal/invoker"
	"github.com/vrealzhou/lambda-local/internal/template"
	"gopkg.in/yaml.v2"
)

var functions map[string]template.Function

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
		tmpl, err := parseTemplate(config.Template())
		if err != nil {
			log.Fatal(err)
		}
		functions = tmpl.Functions()
		defer clearUp()
		serve()
	},
}

func main() {
	parseArgs()
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
	meta := invoker.Functions[name]
	payload, err := ioutil.ReadAll(c.Request().Body)
	defer c.Request().Body.Close()
	if err != nil {
		return err
	}
	result, err := invoker.InvokeFunc(meta, payload)
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
		log.Debugf("Stop function %s, process id: %d", name, f.Pid)
		proc, err := os.FindProcess(f.Pid)
		if err != nil {
			log.Errorf("Error on find process: %d", f.Pid)
		}
		proc.Kill()
	}
}

func parseTemplate(tmplFile string) (t template.SAMTemplate, err error) {
	if tmplFile == "" {
		return t, fmt.Errorf("Please specify template file via --template {filename}")
	}
	f, err := os.Open(tmplFile)
	if err != nil {
		return t, fmt.Errorf("Error when open template file %s: %s", tmplFile, err.Error())
	}
	defer f.Close()
	d := yaml.NewDecoder(f)
	d.SetStrict(false)
	err = d.Decode(&t)
	if err != nil {
		return t, fmt.Errorf("Error when decoding template file %s: %s", tmplFile, err.Error())
	}
	return t, nil
}

func parseArgs() {
	var port int
	rootCmd.PersistentFlags().IntVarP(&port, "port", "p", 3001, "Service port")
	viper.BindPFlag("port", rootCmd.PersistentFlags().Lookup("port"))

	var template string
	rootCmd.PersistentFlags().StringVarP(&template, "template", "t", "/var/lambdas/template.yaml", "SAM template file")
	viper.BindPFlag("template", rootCmd.PersistentFlags().Lookup("template"))

	var lambdaBase string
	rootCmd.PersistentFlags().StringVarP(&lambdaBase, "base", "b", "/var/lambdas", "Lambda base dir")
	viper.BindPFlag("lambdaBase", rootCmd.PersistentFlags().Lookup("base"))

	if os.Getenv("ENV_JSON") == "true" {
		viper.SetDefault("env", "/var/lambdas/env.json")
		os.Unsetenv("ENV_JSON")
	}

	debug := strings.ToLower(os.Getenv("DEBUG")) == "true"
	if debug {
		log.SetLevel(log.DebugLevel)
	}
	log.SetFormatter(&log.TextFormatter{})
}
