package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
	"github.com/vrealzhou/goformation/cloudformation/resources"
	config "github.com/vrealzhou/lambda-local/config/cmd"
	"github.com/vrealzhou/lambda-local/internal/docker"
	"github.com/vrealzhou/lambda-local/internal/template"
)

func main() {
	rootCmd.AddCommand(startLambdaCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(invokeCmd)
	config.ParseArgs(rootCmd, invokeCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Long: "Local lambda invoke service. Require docker installed.",
}

var invokeCmd = &cobra.Command{
	Use:              "invoke [flags] function-name",
	Short:            "Invoke specified function locally",
	TraverseChildren: true,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			fmt.Printf("requires at least one arg\n")
			return
		}
		fmt.Printf("Execute function %s in template %s with reload: %t\n", args[0], config.Template(), config.Reload())
	},
}

var startLambdaCmd = &cobra.Command{
	Use:   "start-lambda",
	Short: "Start local lambda service",
	Long:  "Start local lambda service. You can set Endpoint in your AWS config to invoke this instead cloud env.",
	Run: func(cmd *cobra.Command, args []string) {
		if config.Template() == "" {
			fmt.Printf("argument --template must be set")
			return
		}
		functions := parseTemplate()
		for name, f := range functions {
			fmt.Printf("Function %s: %#v\n", name, f)
		}
		ctx := context.Background()
		cli, err := client.NewEnvClient()
		if err != nil {
			panic(err)
		}
		err = docker.StartLambdaContainer(ctx, cli, functions, config.Parameters())
		if err != nil {
			docker.DeleteContainer(ctx, cli)
			panic(err)
		}
		output := config.Output()
		switch output {
		case "":
			return
		case "STDOUT":
			listenSignal(func() {
				docker.StopContainer(ctx, cli)
			})
			docker.AttachContainer(ctx, cli, os.Stdout)
		default:
			fmt.Printf("Export log to: %s\n", output)
			f, err := os.Create(output)
			if err != nil {
				docker.DeleteContainer(ctx, cli)
				panic(err)
			}
			listenSignal(func() {
				f.Close()
				docker.StopContainer(ctx, cli)
			})
			docker.AttachContainer(ctx, cli, f)
		}
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version",
	Long:  `Print version`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("version 0.1")
	},
}

func parseTemplate() map[string]*resources.AWSServerlessFunction {
	funcs, err := template.Parse(config.Template(), config.Parameters())

	if err != nil {
		panic(err)
	}
	return funcs
}

func listenSignal(f func()) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func(f func()) {
		<-sigs
		f()
	}(f)
}
