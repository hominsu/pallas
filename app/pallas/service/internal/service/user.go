package service

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	v1 "github.com/hominsu/pallas/api/pallas/service/v1"
)

func (s *UserService) Signup(ctx context.Context, in *v1.SignupRequest) (*v1.SignupReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Signup not implemented")
}

func (s *UserService) Signin(ctx context.Context, in *v1.SigninRequest) (*v1.SigninReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Signin not implemented")
}

func (s *UserService) SignOut(ctx context.Context, in *emptypb.Empty) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SignOut not implemented")
}
