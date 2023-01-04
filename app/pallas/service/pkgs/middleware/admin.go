package middleware

import (
	"context"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"

	"github.com/hominsu/pallas/app/pallas/service/internal/data"
)

var (
	ErrNotAdminUser   = errors.Unauthorized(unauthorized, "not admin user")
	ErrMissingUserId  = errors.Unauthorized(unauthorized, "missing userid")
	ErrInternalServer = errors.InternalServer(unauthorized, "internal server error")
)

func Admin(d *data.Default) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			v := ctx.Value("userid")
			if v == nil {
				return nil, ErrMissingUserId
			}
			id, ok := v.(int64)
			if !ok {
				return nil, ErrInternalServer
			}
			_, ok = d.AdminsId[id]
			if !ok {
				return nil, ErrNotAdminUser
			}
			return handler(ctx, req)
		}
	}
}
