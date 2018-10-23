package config

import (
	"github.com/spf13/viper"
)

func Port() int {
	return viper.GetInt("port")
}

func Template() string {
	return viper.GetString("template")
}

func ContainerID() string {
	return viper.GetString("containerID")
}

func SetContainerID(id string) {
	viper.Set("containerID", id)
}

func LambdaBase() string {
	return viper.GetString("lambdaBase")
}

func SetLambdaBase(base string) {
	viper.Set("lambdaBase", base)
}
