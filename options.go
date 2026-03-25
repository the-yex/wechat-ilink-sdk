package ilinksdk

import (
	"log/slog"
	"time"

	"github.com/the-yex/wechat-ilink-sdk/login"
	"github.com/the-yex/wechat-ilink-sdk/middleware"
	"github.com/the-yex/wechat-ilink-sdk/plugin"
)

// Option configures the client using functional options pattern.
type Option func(*Config)

// WithBaseURL sets the API base URL.
func WithBaseURL(url string) Option {
	return func(c *Config) {
		c.BaseURL = url
	}
}

// WithToken sets the authentication token.
func WithToken(token string) Option {
	return func(c *Config) {
		c.Token = token
	}
}

// WithCDNBaseURL sets the CDN base URL.
func WithCDNBaseURL(url string) Option {
	return func(c *Config) {
		c.CDNBaseURL = url
	}
}

// WithTimeout sets the HTTP timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.Timeout = timeout
	}
}

// WithLongPollTimeout sets the long-poll timeout for getUpdates.
func WithLongPollTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.LongPollTimeout = timeout
	}
}

// WithLogger sets the logger.
func WithLogger(logger *slog.Logger) Option {
	return func(c *Config) {
		c.Logger = logger
	}
}

// WithRetry configures retry behavior.
func WithRetry(maxRetries int, waitMin, waitMax time.Duration) Option {
	return func(c *Config) {
		c.MaxRetries = maxRetries
		c.RetryWaitMin = waitMin
		c.RetryWaitMax = waitMax
	}
}

// WithMiddleware adds middleware to the client.
func WithMiddleware(m ...middleware.Middleware) Option {
	return func(c *Config) {
		c.Middleware = append(c.Middleware, m...)
	}
}

// WithPlugins adds plugins to the client.
func WithPlugins(p ...plugin.Plugin) Option {
	return func(c *Config) {
		c.Plugins = append(c.Plugins, p...)
	}
}

// WithTokenStore sets the token store for login persistence.
func WithTokenStore(store login.TokenStore) Option {
	return func(c *Config) {
		c.TokenStore = store
	}
}

// WithOnSessionExpired sets the callback for session expiration.
// When the session expires, this callback is invoked to allow re-login.
// Example:
//
//	client, _ := ilinksdk.NewClient(
//	    ilinksdk.WithTokenStore(tokenStore),
//	    ilinksdk.WithOnSessionExpired(func(ctx context.Context) (*ilink.LoginResult, error) {
//	        fmt.Println("Session expired, please scan QR code to re-login")
//	        return client.Login(ctx, displayQRCode)
//	    }),
//	)
func WithOnSessionExpired(callback SessionExpiredCallback) Option {
	return func(c *Config) {
		c.OnSessionExpired = callback
	}
}