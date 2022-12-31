package service

import (
	"context"

	"google.golang.org/protobuf/types/known/emptypb"

	v1 "github.com/hominsu/pallas/api/pallas/service/v1"
)

func (s *SiteService) Ping(ctx context.Context, req *emptypb.Empty) (*v1.PingReply, error) {
	return &v1.PingReply{
		Version: s.version,
	}, nil
}
