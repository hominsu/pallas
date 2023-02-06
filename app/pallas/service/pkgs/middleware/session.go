package middleware

import (
	"context"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
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
	ContextKeyUserK      ContextKey = "user-srp-k"
	ContextKeyRemoteAddr ContextKey = "remote-addr"
)

type SessionKey string

const (
	SessionKeyUserId SessionKey = "userid"
	SessionKeyUserK  SessionKey = "user-srp-k"
)

var ErrGetSessionStoreFail = errors.Unauthorized(unauthorized, "get session error")

func Session(store *sessions.RedisStore, name string, _ log.Logger) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			if tr, ok := transport.FromServerContext(ctx); ok {
				if ht, ok := tr.(*http.Transport); ok {
					session, err := store.Get(ht, name)
					if err != nil {
						return nil, ErrGetSessionStoreFail
					}
					if userId, ok := session.Values[string(SessionKeyUserId)]; ok {
						if id, ok := userId.(int64); ok {
							ctx = context.WithValue(ctx, ContextKeyUserId, id)
						}
					}
					if userK, ok := session.Values[string(SessionKeyUserK)]; ok {
						if k, ok := userK.([]byte); ok {
							ctx = context.WithValue(ctx, ContextKeyUserK, k)
						}
					}
				}
			}
			return handler(ctx, req)
		}
	}
}
