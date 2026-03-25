// Package ilinksdk provides a Go SDK for WeChat iLink protocol.
package ilinksdk

import (
	"context"
	"log/slog"
	"time"

	"github.com/the-yex/wechat-ilink-sdk/ilink"
	"github.com/the-yex/wechat-ilink-sdk/login"
	"github.com/the-yex/wechat-ilink-sdk/middleware"
	"github.com/the-yex/wechat-ilink-sdk/plugin"
)

// SessionExpiredCallback is called when the session expires.
// Return a LoginResult to continue with a new session, or an error to stop.
type SessionExpiredCallback func(ctx context.Context) (*ilink.LoginResult, error)

// Config holds the SDK configuration.
type Config struct {
	// API configuration
	BaseURL string
	Token   string

	// CDN configuration
	CDNBaseURL string

	// HTTP client settings
	Timeout         time.Duration
	LongPollTimeout time.Duration

	// Retry configuration
	MaxRetries   int
	RetryWaitMin time.Duration
	RetryWaitMax time.Duration

	// Logging
	Logger *slog.Logger

	// Token storage (for login flow)
	TokenStore login.TokenStore

	// Callback when session expires (optional)
	// If set, this will be called when the session expires to allow re-login.
	// Return nil to stop the Run loop, or a LoginResult to continue.
	OnSessionExpired SessionExpiredCallback

	// Extensions
	Middleware []middleware.Middleware
	Plugins    []plugin.Plugin
}

// defaultConfig returns a Config with sensible defaults.
func defaultConfig() *Config {
	return &Config{
		BaseURL:         "https://ilinkai.weixin.qq.com",
		CDNBaseURL:      "https://novac2c.cdn.weixin.qq.com/c2c",
		Timeout:         30 * time.Second,
		LongPollTimeout: 35 * time.Second,
		MaxRetries:      3,
		RetryWaitMin:    1 * time.Second,
		RetryWaitMax:    5 * time.Second,
		Logger:          slog.Default(),
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.BaseURL == "" {
		return ErrInvalidConfig
	}
	return nil
}