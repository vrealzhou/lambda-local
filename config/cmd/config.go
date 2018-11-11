package config

import (
	"github.com/docker/docker/api/types/container"
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

// Profile returns AWS credentials profile name
func Profile() string {
	return viper.GetString("profile")
}

// Payload returns payload file name
func Payload() string {
	return viper.GetString("payload")
}

// SetPayload sets payload file name to global setting
func SetPayload(payload string) {
	viper.Set("payload", payload)
}

// NetworkMode returns docker network mode
func NetworkMode() container.NetworkMode {
	return container.NetworkMode(viper.GetString("networkMode"))
}

// AWSRegion returns AWS Region
func AWSRegion() string {
	return viper.GetString("aws_region")
}

// ContainerID returns containerID
func ContainerID() string {
	return viper.GetString("containerID")
}

// SetContainerID set containerID into global setting
func SetContainerID(id string) {
	viper.Set("containerID", id)
}
