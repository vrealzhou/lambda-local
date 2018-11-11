package config

import (
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
