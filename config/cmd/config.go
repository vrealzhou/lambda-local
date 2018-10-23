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

func Profile() string {
	return viper.GetString("profile")
}

func Payload() string {
	return viper.GetString("payload")
}

func SetPayload(payload string) {
	viper.Set("payload", payload)
}

func ContainerID() string {
	return viper.GetString("containerID")
}

func SetContainerID(id string) {
	viper.Set("containerID", id)
}
