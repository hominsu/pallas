package service

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	v1 "github.com/hominsu/pallas/api/pallas/service/v1"
	"github.com/hominsu/pallas/app/pallas/service/internal/biz"
)

func (s *AdminService) ListUsers(ctx context.Context, req *v1.ListUsersRequest) (*v1.ListUsersReply, error) {
	res, nextPageToken, err := s.uu.ListUsers(
		ctx,
		int(req.GetPageSize()),
		req.GetPageToken(),
		biz.UserView(req.GetView()),
	)
	if err != nil {
		return nil, err
	}
	return &v1.ListUsersReply{
		Users:         res,
		NextPageToken: nextPageToken,
	}, nil
}

func (s *AdminService) CreateGroup(ctx context.Context, req *v1.CreateGroupRequest) (*v1.Group, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateGroup not implemented")
}

func (s *AdminService) GetGroup(ctx context.Context, req *v1.GetGroupRequest) (*v1.Group, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetGroup not implemented")
}

func (s *AdminService) UpdateGroup(ctx context.Context, req *v1.UpdateGroupRequest) (*v1.Group, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateGroup not implemented")
}

func (s *AdminService) DeleteGroup(ctx context.Context, req *v1.DeleteGroupRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteGroup not implemented")
}

func (s *AdminService) ListGroups(ctx context.Context, req *v1.ListGroupsRequest) (*v1.ListGroupsReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListGroups not implemented")
}
