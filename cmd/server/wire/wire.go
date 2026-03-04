//go:build wireinject
// +build wireinject

package wire

import (
	"echoes-api/internal/handler"
	"echoes-api/internal/job"
	"echoes-api/internal/repository"
	"echoes-api/internal/router"
	"echoes-api/internal/server"
	"echoes-api/internal/service"
	"echoes-api/pkg/app"
	"echoes-api/pkg/jwt"
	"echoes-api/pkg/log"
	"echoes-api/pkg/server/http"
	"echoes-api/pkg/sid"
	"echoes-api/pkg/wechat"

	"github.com/google/wire"
	"github.com/spf13/viper"
)

var repositorySet = wire.NewSet(
	repository.NewDB,
	//repository.NewRedis,
	repository.NewRepository,
	repository.NewTransaction,
	repository.NewUserRepository,
)

var serviceSet = wire.NewSet(
	service.NewService,
	service.NewUserService,
)

var handlerSet = wire.NewSet(
	handler.NewHandler,
	handler.NewUserHandler,
	handler.NewWechatHandler,
)

var jobSet = wire.NewSet(
	job.NewJob,
	job.NewUserJob,
)
var serverSet = wire.NewSet(
	server.NewHTTPServer,
	server.NewJobServer,
)

// build App
func newApp(
	httpServer *http.Server,
	jobServer *server.JobServer,
	// task *server.Task,
) *app.App {
	return app.NewApp(
		app.WithServer(httpServer, jobServer),
		app.WithName("auto-go-server"),
	)
}

func NewWire(*viper.Viper, *log.Logger) (*app.App, func(), error) {
	panic(wire.Build(
		repositorySet,
		serviceSet,
		handlerSet,
		jobSet,
		serverSet,
		wire.Struct(new(router.Deps), "*"),
		sid.NewSid,
		jwt.NewJwt,
		wechat.NewWechat,
		newApp,
	))
}
