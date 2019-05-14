package config

import (
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Port returns service port
func Port() int {
	return viper.GetInt("port")
}

// Template returns template file name
func Template() string {
	return viper.GetString("template")
}

// EnvFile returns env file name
func EnvFile() string {
	return viper.GetString("env")
}

// ContainerID returns containerID
func ContainerID() string {
	return viper.GetString("containerID")
}

// SetContainerID set containerID into global setting
func SetContainerID(id string) {
	viper.Set("containerID", id)
}

// LambdaBase returns lambda program base dir
func LambdaBase() string {
	return viper.GetString("lambdaBase")
}

// SetLambdaBase sets lambda program base dir into global setting
func SetLambdaBase(base string) {
	viper.Set("lambdaBase", base)
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

func ParseArgs(rootCmd *cobra.Command) {
	var port int
	rootCmd.PersistentFlags().IntVarP(&port, "port", "", 3001, "Service port")
	viper.BindPFlag("port", rootCmd.PersistentFlags().Lookup("port"))

	var template string
	rootCmd.PersistentFlags().StringVarP(&template, "template", "t", "/var/lambdas/template.yaml", "SAM template file")
	viper.BindPFlag("template", rootCmd.PersistentFlags().Lookup("template"))

	var lambdaBase string
	rootCmd.PersistentFlags().StringVarP(&lambdaBase, "base", "b", "/var/lambdas", "Lambda base dir")
	viper.BindPFlag("lambdaBase", rootCmd.PersistentFlags().Lookup("base"))

	rootCmd.PersistentFlags().StringSliceP("parameter", "p", []string{}, "parameter override")
	viper.BindPFlag("parameter", rootCmd.PersistentFlags().Lookup("parameter"))

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
