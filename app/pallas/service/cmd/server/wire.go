//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"

	"github.com/hominsu/pallas/app/pallas/service/internal/biz"
	"github.com/hominsu/pallas/app/pallas/service/internal/conf"
	"github.com/hominsu/pallas/app/pallas/service/internal/data"
	"github.com/hominsu/pallas/app/pallas/service/internal/server"
	"github.com/hominsu/pallas/app/pallas/service/internal/service"
)

// initApp init kratos application.
func initApp(
	confServer *conf.Server,
	confData *conf.Data,
	confSecret *conf.Secret,
	version string,
	logger log.Logger,
) (*kratos.App, func(), error) {
	panic(
		wire.Build(
			data.ProviderSet,
			biz.ProviderSet,
			service.ProviderSet,
			server.ProviderSet,
			newApp,
		),
	)
}
