package service

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"

	v1 "github.com/hominsu/pallas/api/pallas/service/v1"
	"github.com/hominsu/pallas/app/pallas/service/internal/biz"
	"github.com/hominsu/pallas/pkg/sessions"
)

// ProviderSet is service providers.
var ProviderSet = wire.NewSet(
	NewSiteService,
	NewUserService,
	NewAdminService,
)

type SiteService struct {
	v1.UnimplementedSiteServiceServer

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
	v1.UnimplementedUserServiceServer

	store *sessions.RedisStore
	uu    *biz.UserUsecase
	log   *log.Helper
}

func NewUserService(store *sessions.RedisStore, uu *biz.UserUsecase, logger log.Logger) *UserService {
	return &UserService{
		store: store,
		uu:    uu,
		log:   log.NewHelper(log.With(logger, "module", "service/user")),
	}
}

type AdminService struct {
	v1.UnimplementedAdminServiceServer

	store *sessions.RedisStore
	gu    *biz.GroupUsecase
	uu    *biz.UserUsecase
	log   *log.Helper
}

func NewAdminService(store *sessions.RedisStore, gu *biz.GroupUsecase, uu *biz.UserUsecase, logger log.Logger) *AdminService {
	return &AdminService{
		store: store,
		gu:    gu,
		uu:    uu,
		log:   log.NewHelper(log.With(logger, "module", "service/admin")),
	}
}
