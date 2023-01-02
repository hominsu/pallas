package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"
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
	if tr, ok := transport.FromServerContext(ctx); ok {
		if ht, ok := tr.(*http.Transport); ok {
			res, err := s.uu.Signin(ctx, req.GetEmail(), req.GetPassword())
			if err != nil {
				return nil, err
			}

			session, err := s.store.Get(ht, "pallas-session")
			if err != nil {
				return nil, v1.ErrorSessionError("get session error: %v", err)
			}
			session.Values["userid"] = res.Id
			if err = session.Save(ht); err != nil {
				return nil, v1.ErrorSessionError("save session error: %v", err)
			}

			return &v1.SigninReply{}, nil
		}
	}
	return nil, v1.ErrorInternalError("transport error")
}

func (s *UserService) SignOut(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	if tr, ok := transport.FromServerContext(ctx); ok {
		if ht, ok := tr.(*http.Transport); ok {
			v := ctx.Value("userid")
			if v == nil {
				return nil, v1.ErrorSessionError("session missed")
			}
			id, ok := v.(int64)
			if !ok {
				return nil, v1.ErrorInternalError("transport error")
			}
			session, err := s.store.Get(ht, "pallas-session")
			if err != nil {
				return nil, v1.ErrorSessionError("get session error: %v", err)
			}
			userid, ok := session.Values["userid"].(int64)
			if !ok {
				return nil, v1.ErrorInternalError("transport error")
			}
			if userid != id {
				return nil, v1.ErrorUserMismatch("userid mismatch")
			}
			session.Options.MaxAge = -1
			if err = session.Save(ht); err != nil {
				return nil, v1.ErrorSessionError("save session error: %v", err)
			}
			return &emptypb.Empty{}, nil
		}
	}
	return nil, v1.ErrorInternalError("transport error")
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
