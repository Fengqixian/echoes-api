package router

import (
	"echoes-api/internal/handler"
	"echoes-api/pkg/jwt"
	"echoes-api/pkg/log"

	"github.com/spf13/viper"
)

type Deps struct {
	Logger        *log.Logger
	Config        *viper.Viper
	JWT           *jwt.JWT
	UserHandler   *handler.UserHandler
	WechatHandler *handler.WechatHandler
}
