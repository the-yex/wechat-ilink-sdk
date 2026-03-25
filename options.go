package ilinksdk

import (
	"context"
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

// WithOnLogin sets the callback for QR code login.
// When Run() is called without prior Login(), this callback is used to display QR code.
// Example:
//
//	client, _ := ilinksdk.NewClient(
//	    ilinksdk.WithTokenStore(tokenStore),
//	    ilinksdk.WithOnLogin(func(ctx context.Context, qr *login.QRCode) error {
//	        login.PrintQRCodeWithTerm(qr)
//	        return nil
//	    }),
//	)
//	client.Run(ctx, handler) // Will auto-login if needed
func WithOnLogin(callback login.QRCodeCallback) Option {
	return func(c *Config) {
		c.OnLogin = callback
	}
}

// WithOnLoginSuccess sets the callback for successful login.
// Use this to save login info to your own storage (database, cache, etc.)
// Example:
//
//	client, _ := ilinksdk.NewClient(
//	    ilinksdk.WithOnLoginSuccess(func(ctx context.Context, result *ilink.LoginResult) error {
//	        // Save to database
//	        db.SaveUser(result.AccountID, result.Token, result.UserID)
//	        return nil
//	    }),
//	)
func WithOnLoginSuccess(callback LoginSuccessCallback) Option {
	return func(c *Config) {
		c.OnLoginSuccess = callback
	}
}

// WithTokenProvider sets the provider for loading stored token.
// Use this to load token from your own storage instead of TokenStore.
// Example:
//
//	client, _ := ilinksdk.NewClient(
//	    ilinksdk.WithTokenProvider(func(ctx context.Context) (*login.TokenInfo, error) {
//	        // Load from database
//	        user := db.GetUser(accountID)
//	        if user == nil {
//	            return nil, nil // No token, will trigger login
//	        }
//	        return &login.TokenInfo{Token: user.Token}, nil
//	    }),
//	)
func WithTokenProvider(provider TokenProvider) Option {
	return func(c *Config) {
		c.TokenProvider = provider
	}
}

// WithOnTokenInvalid sets the callback for when token becomes invalid.
// Use this to clear token from your own storage.
// Only works when TokenProvider is set.
// Example:
//
//	client, _ := ilinksdk.NewClient(
//	    ilinksdk.WithTokenProvider(loadTokenFromDB),
//	    ilinksdk.WithOnTokenInvalid(func(ctx context.Context) {
//	        db.DeleteToken(accountID)
//	    }),
//	)
func WithOnTokenInvalid(callback func(ctx context.Context)) Option {
	return func(c *Config) {
		c.OnTokenInvalid = callback
	}
}
