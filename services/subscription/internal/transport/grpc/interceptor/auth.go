package interceptor

import (
	"context"
	"errors"

	"connectrpc.com/connect"
)

func NewAuthInterceptor(apiKey string) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if apiKey == "" {
				return next(ctx, req)
			}

			if req.Header().Get("Authorization") != "Bearer "+apiKey {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid api key"))
			}

			return next(ctx, req)
		}
	}
}
