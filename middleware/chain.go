// Package middleware provides middleware for the WeChat Bot SDK.
package middleware

import (
	"context"

	"github.com/the-yex/wechat-ilink-sdk/ilink"
)

// Handler is the function signature for message handlers.
type Handler func(ctx context.Context, req *ilink.SendMessageRequest) error

// Middleware wraps a Handler with additional functionality.
type Middleware func(next Handler) Handler

// Chain creates a middleware chain from multiple middlewares.
// Middlewares are applied in reverse order so that the first
// middleware in the list is the outermost wrapper.
func Chain(final Handler, middlewares ...Middleware) Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		final = middlewares[i](final)
	}
	return final
}