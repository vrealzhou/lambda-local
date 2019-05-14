package config

import (
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// AWSRegion returns AWS Region
func AWSRegion() string {
	return viper.GetString("aws_region")
}

// ContainerID returns containerID
func ContainerID() string {
	return viper.GetString("containerID")
}

// Env returns env file name
func Env() map[string]string {
	envAry := viper.GetStringSlice("env")
	envMap := make(map[string]string)
	for _, item := range envAry {
		index := strings.Index(item, "=")
		envMap[item[:index]] = item[index+1:]
	}
	return envMap
}

// Parameters returns cloud formation parameter overrides
func Parameters() map[string]string {
	ary := viper.GetStringSlice("parameter")
	m := make(map[string]string)
	for _, item := range ary {
		index := strings.Index(item, "=")
		m[item[:index]] = item[index+1:]
	}
	return m
}

// EnvFile returns env file name
func EnvFile() string {
	return viper.GetString("env-json")
}

// NetworkMode returns docker network mode
func NetworkMode() container.NetworkMode {
	return container.NetworkMode(viper.GetString("networkMode"))
}

// Output returns output direction: STDOUT or filename
func Output() string {
	return viper.GetString("output")
}

// Payload returns payload file name
func Payload() string {
	return viper.GetString("payload")
}

// Port returns service port
func Port() int {
	return viper.GetInt("port")
}

// Profile returns AWS credentials profile name
func Profile() string {
	return viper.GetString("profile")
}

// Template returns template file name
func Template() string {
	return viper.GetString("template")
}

// SetContainerID set containerID into global setting
func SetContainerID(id string) {
	viper.Set("containerID", id)
}

// SetPayload sets payload file name to global setting
func SetPayload(payload string) {
	viper.Set("payload", payload)
}

// Reload returns true or false of reload
func Reload() bool {
	return viper.GetBool("reload")
}

func ParseArgs(rootCmd, invokeCmd *cobra.Command) {
	var port int
	rootCmd.PersistentFlags().IntVarP(&port, "port", "", 3001, "Service port")
	viper.BindPFlag("port", rootCmd.PersistentFlags().Lookup("port"))

	var profile string
	rootCmd.PersistentFlags().StringVar(&profile, "profile", "default", "AWS credential profile name")
	viper.BindPFlag("profile", rootCmd.PersistentFlags().Lookup("profile"))

	var template string
	rootCmd.PersistentFlags().StringVarP(&template, "template", "t", "", "SAM template file")
	rootCmd.MarkFlagRequired("template")
	viper.BindPFlag("template", rootCmd.PersistentFlags().Lookup("template"))

	var network string
	rootCmd.PersistentFlags().StringVarP(&network, "docker-network", "n", "bridge", "Docker network mode")
	viper.BindPFlag("networkMode", rootCmd.PersistentFlags().Lookup("docker-network"))

	var envjson string
	rootCmd.PersistentFlags().StringVarP(&envjson, "env-json", "", "", "Env json file")
	viper.BindPFlag("env-json", rootCmd.PersistentFlags().Lookup("env-json"))

	var awsRegion string
	rootCmd.PersistentFlags().StringVarP(&awsRegion, "aws-region", "r", "", "AWS region")
	viper.BindPFlag("aws_region", rootCmd.PersistentFlags().Lookup("aws-region"))

	var reload bool
	invokeCmd.Flags().BoolVar(&reload, "reload", true, "reload lambda")
	viper.BindPFlag("reload", invokeCmd.Flags().Lookup("reload"))

	var debug bool
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Turn on/off debug")
	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))

	rootCmd.PersistentFlags().StringSliceP("env", "e", []string{}, "env settings")
	viper.BindPFlag("env", rootCmd.PersistentFlags().Lookup("env"))

	rootCmd.PersistentFlags().StringSliceP("parameter", "p", []string{}, "parameter override")
	viper.BindPFlag("parameter", rootCmd.PersistentFlags().Lookup("parameter"))

	rootCmd.PersistentFlags().StringP("output", "o", "", `output targets: STDOUT, filename. 
	It will stop command line and let container keep running if empty`)
	viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output"))
}
