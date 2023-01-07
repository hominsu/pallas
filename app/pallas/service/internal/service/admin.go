package service

import (
	"context"

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
	group, err := biz.ToGroup(req.GetGroup())
	if err != nil {
		return nil, err
	}
	res, err := s.gu.CreateGroup(ctx, group)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (s *AdminService) GetGroup(ctx context.Context, req *v1.GetGroupRequest) (*v1.Group, error) {
	res, err := s.gu.GetGroup(ctx, req.GetId(), biz.GroupView(req.GetView()))
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (s *AdminService) UpdateGroup(ctx context.Context, req *v1.UpdateGroupRequest) (*v1.Group, error) {
	group, err := biz.ToGroup(req.GetGroup())
	if err != nil {
		return nil, err
	}

	res, err := s.gu.UpdateGroup(ctx, group)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *AdminService) DeleteGroup(ctx context.Context, req *v1.DeleteGroupRequest) (*emptypb.Empty, error) {
	err := s.gu.DeleteGroup(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *AdminService) ListGroups(ctx context.Context, req *v1.ListGroupsRequest) (*v1.ListGroupsReply, error) {
	res, nextPageToken, err := s.gu.ListGroups(
		ctx,
		int(req.GetPageSize()),
		req.GetPageToken(),
		biz.GroupView(req.GetView()),
	)
	if err != nil {
		return nil, err
	}
	return &v1.ListGroupsReply{
		GroupList:     res,
		NextPageToken: nextPageToken,
	}, nil
}
