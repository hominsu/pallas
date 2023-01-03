package middleware

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"

	"github.com/hominsu/pallas/pkg/sessions"
)

func Session(store *sessions.RedisStore, name string, helper *log.Helper) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			if tr, ok := transport.FromServerContext(ctx); ok {
				if ht, ok := tr.(*http.Transport); ok {
					session, err := store.Get(ht, name)
					if err != nil {
						helper.Error(fmt.Sprintf("get session error: %v", err))
						return handler(ctx, req)
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
