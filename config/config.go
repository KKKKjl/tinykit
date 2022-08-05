package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// app default value.
const (
	_httpPort  = "8000"
	_grpcPort  = "8080"
	_envPrefix = "tinykit"
)

func InitConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config/")
	viper.AddConfigPath(".")

	viper.Set("Verbose", true)

	addDefault()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			panic(fmt.Errorf("config not found: %v", err))
		} else {
			panic(fmt.Errorf("config error: %v", err))
		}
	}
}

func addDefault() {
	// app
	defaultVar("HTTP_PORT", _httpPort)
	defaultVar("GRPC_PORT", _grpcPort)
	defaultVar("SIGNING_AlGORITHM", "HS256")

	// env
	viper.SetEnvPrefix(_envPrefix)
	viper.BindEnv("jwt_secret")
}

func defaultVar(key string, value interface{}) string {
	viper.SetDefault(key, value)
	return key
}
