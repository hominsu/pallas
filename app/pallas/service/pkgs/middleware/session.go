package middleware

import (
	"context"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"

	v1 "github.com/hominsu/pallas/api/pallas/service/v1"
	"github.com/hominsu/pallas/pkg/redisstore"
)

func Session(store *redisstore.RedisStore, name string) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			var userId int64
			if tr, ok := transport.FromServerContext(ctx); ok {
				if ht, ok := tr.(*http.Transport); ok {
					session, err := store.Get(ht.Request(), name)
					if err != nil {
						return nil, v1.ErrorSessionError("get session error: %v", err)
					}
					if id, ok := session.Values["x-md-global-userid"]; ok {
						if userId, ok = id.(int64); ok {
							ctx = context.WithValue(ctx, "x-md-global-userid", userId)
						}
					}
				}
			}
			return handler(ctx, req)
		}
	}
}
