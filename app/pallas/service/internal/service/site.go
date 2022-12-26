package service

import (
	"context"

	siteV1 "github.com/hominsu/pallas/api/pallas/service/v1"
)

func (s *SiteService) Ping(context.Context, *siteV1.PingRequest) (*siteV1.PingReply, error) {
	return &siteV1.PingReply{
		Version: "1.0.0",
	}, nil
}
