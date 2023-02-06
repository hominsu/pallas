package middleware

import (
	"context"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"

	"github.com/hominsu/pallas/app/pallas/service/internal/biz"
)

var (
	ErrNotAdminUser   = errors.Unauthorized(unauthorized, "not admin user")
	ErrMissingUserId  = errors.Unauthorized(unauthorized, "missing userid")
	ErrInternalServer = errors.InternalServer(unauthorized, "internal server error")
)

func Admin(uu *biz.UserUsecase, logger log.Logger) middleware.Middleware {
	helper := log.NewHelper(log.With(logger, "module", "middleware/admin"))

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			// get user id from context
			v := ctx.Value(ContextKeyUserId)
			if v == nil {
				return nil, ErrMissingUserId
			}
			id, ok := v.(int64)
			if !ok {
				helper.Warnf("failed to get userid from context")
				return nil, ErrInternalServer
			}

			ok, err := uu.IsAdminUser(ctx, id)
			if err != nil {
				helper.Warnf("failed to check user if is admin user, err: %v", err)
				return nil, ErrInternalServer
			}
			if !ok {
				return nil, ErrNotAdminUser
			}

			return handler(ctx, req)
		}
	}
}
