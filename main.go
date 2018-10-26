package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	config "github.com/vrealzhou/lambda-local/config/cmd"
	"github.com/vrealzhou/lambda-local/internal/docker"
	"github.com/vrealzhou/lambda-local/internal/template"
	"gopkg.in/yaml.v2"
)

func main() {
	rootCmd.AddCommand(startLambdaCmd)
	rootCmd.AddCommand(versionCmd)
	parseArgs()
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Long: "Local lambda invoke service. Require docker installed.",
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
		template := parseTemplate()
		functions := template.Functions()
		for name, f := range functions {
			fmt.Printf("Function %s: %#v\n", name, f)
		}
		ctx := context.Background()
		cli, err := client.NewEnvClient()
		if err != nil {
			panic(err)
		}
		err = docker.StartLambdaContainer(ctx, cli, functions)
		if err != nil {
			docker.DeleteContainer(ctx, cli)
			panic(err)
		}
		listenSignal(ctx, cli)
		docker.AttachContainer(ctx, cli)
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

func parseTemplate() template.SAMTemplate {
	f, err := os.Open(config.Template())
	if err != nil {
		panic(err)
	}
	defer f.Close()
	d := yaml.NewDecoder(f)
	d.SetStrict(false)
	template := template.SAMTemplate{}
	err = d.Decode(&template)
	if err != nil {
		panic(err)
	}
	return template
}

func parseArgs() {
	var port int
	var profile string
	var template string
	var network string
	var awsRegion string
	var debug bool
	rootCmd.PersistentFlags().IntVarP(&port, "port", "p", 3001, "Service port")
	viper.BindPFlag("port", rootCmd.PersistentFlags().Lookup("port"))
	rootCmd.PersistentFlags().StringVar(&profile, "profile", "default", "AWS credential profile name")
	viper.BindPFlag("profile", rootCmd.PersistentFlags().Lookup("profile"))
	rootCmd.PersistentFlags().StringVarP(&template, "template", "t", "", "SAM template file")
	rootCmd.MarkFlagRequired("template")
	viper.BindPFlag("template", rootCmd.PersistentFlags().Lookup("template"))
	rootCmd.PersistentFlags().StringVarP(&network, "docker-network", "n", "bridge", "Docker network mode")
	viper.BindPFlag("networkMode", rootCmd.PersistentFlags().Lookup("docker-network"))
	rootCmd.PersistentFlags().StringVarP(&awsRegion, "aws-region", "r", "", "AWS region")
	viper.BindPFlag("aws_region", rootCmd.PersistentFlags().Lookup("aws-region"))
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Turn on/off debug")
	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
}

func listenSignal(ctx context.Context, cli *client.Client) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func(ctx context.Context, cli *client.Client) {
		<-sigs
		docker.StopContainer(ctx, cli)
	}(ctx, cli)
}
