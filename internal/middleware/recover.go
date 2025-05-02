package middleware

import (
	"context"
	"go.uber.org/zap"
	"runtime/debug"
)

func RecoverMiddleware(next func(context.Context, []byte) error, logger *zap.Logger) func(context.Context, []byte) error {
	return func(ctx context.Context, data []byte) error {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic recovered",
					zap.Any("panic", r),
					zap.String("stack", string(debug.Stack())))
			}
		}()
		return next(ctx, data)
	}
}
