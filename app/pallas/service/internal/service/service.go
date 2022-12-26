package service

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"

	siteV1 "github.com/hominsu/pallas/api/pallas/service/v1"
)

// ProviderSet is service providers.
var ProviderSet = wire.NewSet(NewSiteService)

type SiteService struct {
	siteV1.UnimplementedSiteServer

	log *log.Helper
}

func NewSiteService(logger log.Logger) *SiteService {
	return &SiteService{
		log: log.NewHelper(log.With(logger, "module", "service/site")),
	}
}
