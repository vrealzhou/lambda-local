package config

import (
	"strings"

	"github.com/docker/docker/api/types/container"
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
	envAry := viper.Get("env").([]string)
	envMap := make(map[string]string)
	for _, item := range envAry {
		index := strings.Index(item, "=")
		envMap[item[:index]] = item[index+1:]
	}
	return envMap
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
