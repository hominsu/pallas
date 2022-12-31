package service

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	v1 "github.com/hominsu/pallas/api/pallas/service/v1"
)

func (s *UserService) Signup(ctx context.Context, req *v1.SignupRequest) (*v1.User, error) {
	res, err := s.uu.Signup(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (s *UserService) Signin(ctx context.Context, req *v1.SigninRequest) (*v1.SigninReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Signin not implemented")
}

func (s *UserService) SignOut(ctx context.Context, req *emptypb.Empty) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SignOut not implemented")
}

func (s *UserService) GetUser(ctx context.Context, req *v1.GetUserRequest) (*v1.User, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetUser not implemented")
}

func (s *UserService) UpdateUser(ctx context.Context, req *v1.UpdateUserRequest) (*v1.User, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateUser not implemented")
}

func (s *UserService) DeleteUser(ctx context.Context, req *v1.DeleteUserRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteUser not implemented")
}

func (s *UserService) ListUsers(ctx context.Context, req *v1.ListUsersRequest) (*v1.ListUsersReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListUsers not implemented")
}
