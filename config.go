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

// LoginSuccessCallback is called after successful login.
// Users can save the login result to their own storage.
type LoginSuccessCallback func(ctx context.Context, result *ilink.LoginResult) error

// TokenProvider is called when SDK needs token info.
// Return stored token info, or nil if not available.
type TokenProvider func(ctx context.Context) (*login.TokenInfo, error)

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

	// Rate limiting for outbound message sends.
	RateLimitMessagesPerSecond int
	RateLimitBurst             int

	// Logging
	Logger *slog.Logger

	// Token storage (for auto-login) - default: FileTokenStore in ./.weixin/
	// If you want to manage tokens yourself, use OnLoginSuccess and TokenProvider instead.
	TokenStore login.TokenStore

	// Login callback - called when QR code login is needed
	// Default: displays QR code in terminal using login.PrintQRCodeWithTerm
	// Set this only if you want custom QR code display (e.g., web UI)
	OnLogin login.QRCodeCallback

	// Callback when session expires
	// Default: automatically prompts for QR code re-scan via login.PrintQRCodeWithTerm
	// Set this only if you want custom handling (e.g., stop the loop, notify monitoring)
	// Return nil to stop the Run loop, or a LoginResult to continue.
	OnSessionExpired SessionExpiredCallback

	// OnLoginSuccess is called after successful login.
	// Use this to save login info to your own storage (database, cache, etc.)
	// Example: save to database for multi-account support
	OnLoginSuccess LoginSuccessCallback

	// TokenProvider is called when SDK needs stored token info.
	// Use this to load token from your own storage.
	// Return nil if no token stored (will trigger login flow).
	TokenProvider TokenProvider

	// OnTokenInvalid is called when token becomes invalid (expired, session timeout, etc.)
	// Use this to clear stored token from your own storage.
	// Only called if TokenProvider is set (not for TokenStore).
	OnTokenInvalid func(ctx context.Context)

	// Extensions
	Middleware []middleware.Middleware
	Plugins    []plugin.Plugin

	// Internal flags for auto-wiring middleware configured via options.
	autoRetry     bool
	autoRateLimit bool
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
		RateLimitBurst:  1,
		Logger:          slog.Default(),
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.BaseURL == "" {
		return ErrInvalidConfig
	}
	if c.autoRetry {
		if c.MaxRetries < 1 || c.RetryWaitMin <= 0 || c.RetryWaitMax <= 0 || c.RetryWaitMax < c.RetryWaitMin {
			return ErrInvalidConfig
		}
	}
	if c.autoRateLimit {
		if c.RateLimitMessagesPerSecond < 1 || c.RateLimitBurst < 1 {
			return ErrInvalidConfig
		}
	}
	return nil
}

// buildMiddleware returns the resolved middleware chain, including any middleware
// configured through high-level options such as WithRetry and WithRateLimit.
func buildMiddleware(c *Config) []middleware.Middleware {
	chain := make([]middleware.Middleware, 0, len(c.Middleware)+2)
	chain = append(chain, c.Middleware...)

	// Retry wraps the transport first so each retry attempt still passes through
	// downstream middleware such as rate limiting.
	if c.autoRetry {
		chain = append(chain, middleware.Retry(middleware.RetryConfig{
			MaxAttempts: c.MaxRetries,
			WaitMin:     c.RetryWaitMin,
			WaitMax:     c.RetryWaitMax,
			Retryable:   middleware.DefaultRetryable,
		}))
	}

	if c.autoRateLimit {
		chain = append(chain, middleware.RateLimit(middleware.RateLimitConfig{
			MessagesPerSecond: c.RateLimitMessagesPerSecond,
			Burst:             c.RateLimitBurst,
		}))
	}

	return chain
}
