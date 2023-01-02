package middleware

import (
	"context"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"

	v1 "github.com/hominsu/pallas/api/pallas/service/v1"
	"github.com/hominsu/pallas/pkg/sessions"
)

func Session(store *sessions.RedisStore, name string) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			if tr, ok := transport.FromServerContext(ctx); ok {
				if ht, ok := tr.(*http.Transport); ok {
					session, err := store.Get(ht, name)
					if err != nil {
						return nil, v1.ErrorSessionError("get session error: %v", err)
					}
					if id, ok := session.Values["userid"]; ok {
						if userId, ok := id.(int64); ok {
							ctx = context.WithValue(ctx, "userid", userId)
						}
					}
				}
			}
			return handler(ctx, req)
		}
	}
}
