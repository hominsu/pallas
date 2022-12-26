package service

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"

	v1 "github.com/hominsu/pallas/api/pallas/service/v1"
)

// ProviderSet is service providers.
var ProviderSet = wire.NewSet(NewSiteService, NewUserService)

type SiteService struct {
	v1.UnimplementedSiteServer

	version string
	log     *log.Helper
}

func NewSiteService(version string, logger log.Logger) *SiteService {
	return &SiteService{
		version: version,
		log:     log.NewHelper(log.With(logger, "module", "service/site")),
	}
}

type UserService struct {
	v1.UnimplementedUserServer

	log *log.Helper
}

func NewUserService(logger log.Logger) *UserService {
	return &UserService{
		log: log.NewHelper(log.With(logger, "module", "service/user")),
	}
}
