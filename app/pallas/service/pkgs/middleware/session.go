package middleware

import (
	"context"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"

	"github.com/hominsu/pallas/pkg/sessions"
)

const (
	unauthorized string = "UNAUTHORIZED"
)

type ContextKey string

const (
	ContextKeyUserId     ContextKey = "userid"
	ContextKeyRemoteAddr ContextKey = "remote-addr"
)

type SessionKey string

const (
	SessionKeyUserId SessionKey = "userid"
)

var ErrGetSessionStoreFail = errors.Unauthorized(unauthorized, "get session error")

func Session(store *sessions.RedisStore, name string) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			if tr, ok := transport.FromServerContext(ctx); ok {
				if ht, ok := tr.(*http.Transport); ok {
					session, err := store.Get(ht, name)
					if err != nil {
						return nil, ErrGetSessionStoreFail
					}
					if id, ok := session.Values[SessionKeyUserId]; ok {
						if userId, ok := id.(int64); ok {
							ctx = context.WithValue(ctx, ContextKeyUserId, userId)
						}
					}
				}
			}
			return handler(ctx, req)
		}
	}
}
