package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"
	"google.golang.org/protobuf/types/known/emptypb"

	v1 "github.com/hominsu/pallas/api/pallas/service/v1"
	"github.com/hominsu/pallas/app/pallas/service/internal/biz"
	"github.com/hominsu/pallas/app/pallas/service/pkgs/middleware"
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
			session.Values[string(middleware.SessionKeyUserId)] = res.Id
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
			session, err := s.store.Get(ht, "pallas-session")
			if err != nil {
				return nil, v1.ErrorSessionError("get session error: %v", err)
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
	if err := checkUserId(ctx, req.GetId()); err != nil {
		return nil, err
	}
	res, err := s.uu.GetUser(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (s *UserService) UpdateUser(ctx context.Context, req *v1.UpdateUserRequest) (*v1.User, error) {
	if err := checkUserId(ctx, req.GetUser().GetId()); err != nil {
		return nil, err
	}
	user, err := biz.ToUser(req.GetUser())
	if err != nil {
		return nil, err
	}
	res, err := s.uu.UpdateUser(ctx, user)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (s *UserService) DeleteUser(ctx context.Context, req *v1.DeleteUserRequest) (*emptypb.Empty, error) {
	if err := checkUserId(ctx, req.GetId()); err != nil {
		return nil, err
	}
	if err := s.uu.DeleteUser(ctx, req.GetId()); req != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func getUserId(ctx context.Context) (int64, error) {
	v := ctx.Value(middleware.ContextKeyUserId)
	if v == nil {
		return 0, v1.ErrorSessionError("session missed")
	}
	id, ok := v.(int64)
	if !ok {
		return 0, v1.ErrorInternalError("internal error")
	}
	return id, nil
}

func checkUserId(ctx context.Context, userId int64) error {
	id, err := getUserId(ctx)
	if err != nil {
		return err
	}
	if userId != id {
		return v1.ErrorUserMismatch("userid mismatch")
	}
	return nil
}
