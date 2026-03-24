package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/the-yex/wechat-ilink-sdk/ilink"
)

// Logging returns a logging middleware.
func Logging(logger *slog.Logger) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, req *ilink.SendMessageRequest) error {
			start := time.Now()

			to := ""
			msgType := ilink.MessageTypeNone
			if req.Message != nil {
				to = req.Message.ToUserID
				msgType = req.Message.MessageType
			}

			logger.Debug("sending message",
				"to", to,
				"type", msgType,
			)

			err := next(ctx, req)

			duration := time.Since(start)
			if err != nil {
				logger.Error("message send failed",
					"to", to,
					"duration", duration,
					"error", err,
				)
			} else {
				logger.Info("message sent",
					"to", to,
					"duration", duration,
				)
			}

			return err
		}
	}
}