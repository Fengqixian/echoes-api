package config

import (
	"echoes-api/docs"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

func NewConfig(p string) *viper.Viper {
	envConf := os.Getenv("APP_CONF")

	if strings.Contains(envConf, "prod") {
		docs.SwaggerInfo.Host = "www.hookfunc.com/rpa"
	} else {
		docs.SwaggerInfo.Host = "localhost:8000"
	}
	if envConf == "" {
		envConf = p
	}
	fmt.Println("load conf file:", envConf)
	return getConfig(envConf)
}

func getConfig(path string) *viper.Viper {
	conf := viper.New()
	conf.SetConfigFile(path)
	err := conf.ReadInConfig()
	if err != nil {
		panic(err)
	}
	return conf
}
