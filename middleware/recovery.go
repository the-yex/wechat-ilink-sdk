package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"

	"github.com/the-yex/wechat-ilink-sdk/ilink"
)

// Recovery returns a middleware that recovers from panics.
func Recovery(logger *slog.Logger) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, req *ilink.SendMessageRequest) (err error) {
			defer func() {
				if r := recover(); r != nil {
					stack := debug.Stack()
					logger.Error("panic recovered",
						"panic", r,
						"stack", string(stack),
					)
					err = &PanicError{Value: r, Stack: stack}
				}
			}()
			return next(ctx, req)
		}
	}
}

// PanicError represents a recovered panic.
type PanicError struct {
	Value interface{}
	Stack []byte
}

func (e *PanicError) Error() string {
	return fmt.Sprintf("panic: %v", e.Value)
}
