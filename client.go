package ilinksdk

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/the-yex/wechat-ilink-sdk/event"
	"github.com/the-yex/wechat-ilink-sdk/ilink"
	"github.com/the-yex/wechat-ilink-sdk/internal/contextmgr"
	"github.com/the-yex/wechat-ilink-sdk/login"
	"github.com/the-yex/wechat-ilink-sdk/media"
	"github.com/the-yex/wechat-ilink-sdk/middleware"
	"github.com/the-yex/wechat-ilink-sdk/plugin"
	"github.com/the-yex/wechat-ilink-sdk/service"
)

// ContextTokenManager is an alias for contextmgr.ContextTokenManager
type ContextTokenManager = contextmgr.ContextTokenManager

// NewContextTokenManager creates a new context token manager.
func NewContextTokenManager() *ContextTokenManager {
	return contextmgr.NewContextTokenManager()
}

// Client is the main entry point for the WeChat Bot SDK.
// It acts as a facade that delegates to specialized services.
type Client struct {
	config *Config

	// Services
	messages service.MessageService
	media    service.MediaService
	auth     service.AuthService
	session  service.SessionService

	// Shared resources (kept for internal use and backward compatibility)
	apiClient     *ilink.Client
	cdnClient     *media.Client
	contextTokens *ContextTokenManager
	tokenStore    login.TokenStore

	// Extensions
	plugins    *plugin.Registry
	middleware []middleware.Middleware
	events     *event.Dispatcher

	// Polling state
	mu       sync.Mutex
	running  bool
	stopChan chan struct{}
}

// NewClient creates a new WeChat Bot client with the given options.
// Token is optional - if not provided, use Login() to authenticate.
func NewClient(opts ...Option) (*Client, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Create API client
	apiClient := ilink.NewClient(ilink.ClientConfig{
		BaseURL:         cfg.BaseURL,
		Token:           cfg.Token,
		Timeout:         cfg.Timeout,
		LongPollTimeout: cfg.LongPollTimeout,
	})

	// Create CDN client
	cdnClient := media.NewClient(cfg.CDNBaseURL, apiClient)

	// Default token store if not provided
	tokenStore := cfg.TokenStore
	if tokenStore == nil {
		tokenStore = login.NewMemoryTokenStore()
	}

	// Context token manager
	contextTokens := NewContextTokenManager()

	// Create client struct
	client := &Client{
		config:        cfg,
		apiClient:     apiClient,
		cdnClient:     cdnClient,
		contextTokens: contextTokens,
		tokenStore:    tokenStore,
		middleware:    cfg.Middleware,
		events:        event.NewDispatcher(),
		stopChan:      make(chan struct{}),
	}

	// Initialize services
	client.messages = service.NewMessageService(apiClient, cdnClient, contextTokens, cfg.Middleware)
	client.media = service.NewMediaService(cdnClient)
	client.session = service.NewSessionService(apiClient)
	client.auth = service.NewAuthService(
		apiClient,
		cdnClient,
		tokenStore,
		&service.AuthConfig{
			BaseURL:         cfg.BaseURL,
			CDNBaseURL:      cfg.CDNBaseURL,
			Token:           cfg.Token,
			Timeout:         cfg.Timeout,
			LongPollTimeout: cfg.LongPollTimeout,
		},
		client.onTokenUpdate, // Callback for token updates
	)

	// Initialize plugin registry with SDK interface
	client.plugins = plugin.NewRegistry(client)

	// Register initial plugins
	for _, p := range cfg.Plugins {
		if err := client.plugins.Register(p); err != nil {
			return nil, fmt.Errorf("register plugin %s: %w", p.Name(), err)
		}
	}

	return client, nil
}

// onTokenUpdate handles token updates from AuthService.
// It recreates the apiClient and cdnClient with the new token.
func (c *Client) onTokenUpdate(token, baseURL string) {
	if baseURL != "" {
		c.config.BaseURL = baseURL
	}

	// Recreate API client with new token
	c.apiClient = ilink.NewClient(ilink.ClientConfig{
		BaseURL:         c.config.BaseURL,
		Token:           token,
		Timeout:         c.config.Timeout,
		LongPollTimeout: c.config.LongPollTimeout,
	})

	// Update CDN client reference
	c.cdnClient = media.NewClient(c.config.CDNBaseURL, c.apiClient)

	// Update services with new clients
	c.messages = service.NewMessageService(c.apiClient, c.cdnClient, c.contextTokens, c.middleware)
	c.media = service.NewMediaService(c.cdnClient)
	c.session = service.NewSessionService(c.apiClient)
}

// Run starts the message polling loop and processes messages with the given handler.
// This is a blocking call. Use context cancellation to stop.
func (c *Client) Run(ctx context.Context, handler MessageHandler) error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return fmt.Errorf("client is already running")
	}
	c.running = true
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		c.running = false
		c.mu.Unlock()
		// Dispatch disconnected event
		c.events.Dispatch(ctx, &event.Event{
			Type:    event.EventTypeDisconnected,
			Context: ctx,
		})
	}()

	// Initialize plugins
	if err := c.plugins.Initialize(ctx); err != nil {
		return fmt.Errorf("initialize plugins: %w", err)
	}

	// Dispatch connected event
	c.events.Dispatch(ctx, &event.Event{
		Type:    event.EventTypeConnected,
		Context: ctx,
	})

	// Get updates buffer
	var getUpdatesBuf string

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.stopChan:
			return nil
		default:
		}

		// Check if session is paused
		if c.apiClient.IsPaused() {
			// Dispatch session expired event
			c.events.Dispatch(ctx, &event.Event{
				Type:    event.EventTypeSessionExpired,
				Context: ctx,
			})
			c.config.Logger.Warn("session paused, waiting",
				"remaining", c.apiClient.RemainingPause(),
			)
			continue
		}

		// Long poll for messages
		resp, err := c.apiClient.GetUpdates(ctx, &ilink.GetUpdatesRequest{
			GetUpdatesBuf: getUpdatesBuf,
		})
		if err != nil {
			// Dispatch error event
			c.events.Dispatch(ctx, &event.Event{
				Type:    event.EventTypeError,
				Data:    err,
				Context: ctx,
			})
			c.plugins.OnError(ctx, err)
			c.config.Logger.Error("get updates failed", "error", err)
			continue
		}

		// Update buffer for next request
		if resp.GetUpdatesBuf != "" {
			getUpdatesBuf = resp.GetUpdatesBuf
		}

		// Process messages
		for _, msg := range resp.Messages {
			// Store context token
			if msg.ContextToken != "" && msg.FromUserID != "" {
				c.contextTokens.Set(msg.ToUserID, msg.FromUserID, msg.ContextToken)
			}

			// Dispatch message event
			c.events.Dispatch(ctx, &event.Event{
				Type:    event.EventTypeMessage,
				Data:    msg,
				Context: ctx,
			})

			// Process through plugins first
			if err := c.plugins.OnMessage(ctx, msg); err != nil {
				c.plugins.OnError(ctx, err)
				continue
			}

			// Call user handler
			if handler != nil {
				if err := handler(ctx, msg); err != nil {
					c.plugins.OnError(ctx, err)
					c.config.Logger.Error("handler error", "error", err)
				}
			}
		}
	}
}

// MessageHandler handles received messages.
type MessageHandler func(ctx context.Context, msg *ilink.Message) error

// --- MessageService delegation ---

// SendMessage sends a message.
func (c *Client) SendMessage(ctx context.Context, req *ilink.SendMessageRequest) error {
	return c.messages.SendMessage(ctx, req)
}

// SendText sends a text message.
func (c *Client) SendText(ctx context.Context, toUserID, text string) error {
	return c.messages.SendText(ctx, toUserID, text)
}

// SendImage sends an image message.
func (c *Client) SendImage(ctx context.Context, toUserID string, imageData []byte) error {
	return c.messages.SendImage(ctx, toUserID, imageData)
}

// SendTyping sends a typing indicator.
func (c *Client) SendTyping(ctx context.Context, toUserID string, typing bool) error {
	return c.messages.SendTyping(ctx, toUserID, typing)
}

// --- MediaService delegation ---

// UploadMedia uploads a media file to CDN.
func (c *Client) UploadMedia(ctx context.Context, req *media.UploadRequest) (*media.UploadResult, error) {
	return c.media.Upload(ctx, req)
}

// DownloadMedia downloads and decrypts a media file from CDN.
func (c *Client) DownloadMedia(ctx context.Context, req *media.DownloadRequest) ([]byte, error) {
	return c.media.Download(ctx, req)
}

// --- AuthService delegation ---

// Login performs QR code login and returns the login result.
// The displayCallback is called with context and the QR code for display.
func (c *Client) Login(ctx context.Context, displayCallback login.QRCodeCallback) (*ilink.LoginResult, error) {
	result, err := c.auth.Login(ctx, displayCallback)
	if err != nil {
		return nil, err
	}

	// Dispatch login event
	c.events.Dispatch(ctx, &event.Event{
		Type:    event.EventTypeLogin,
		Data:    result,
		Context: ctx,
	})

	return result, nil
}

// LoginSimple performs QR code login with a simple callback.
// Deprecated: Use Login(ctx, callback) with login.QRCodeCallback instead.
func (c *Client) LoginSimple(ctx context.Context, displayCallback func(qr *login.QRCode) error) (*ilink.LoginResult, error) {
	return c.Login(ctx, func(_ context.Context, qr *login.QRCode) error {
		return displayCallback(qr)
	})
}

// SetToken sets the authentication token.
func (c *Client) SetToken(token, baseURL string) {
	c.auth.SetToken(token, baseURL)
}

// LoadToken loads a stored token for an account.
func (c *Client) LoadToken(accountID string) error {
	return c.auth.LoadToken(accountID)
}

// ListAccounts lists all stored account IDs.
func (c *Client) ListAccounts() ([]string, error) {
	return c.auth.ListAccounts()
}

// --- SessionService delegation ---

// IsPaused returns true if the session is paused.
func (c *Client) IsPaused() bool {
	return c.session.IsPaused()
}

// RemainingPause returns the remaining pause duration.
func (c *Client) RemainingPause() time.Duration {
	return c.session.RemainingPause()
}

// --- Middleware and Plugin management ---

// Use adds middleware to the client.
func (c *Client) Use(m ...middleware.Middleware) {
	c.middleware = append(c.middleware, m...)
	// Update MessageService with new middleware
	c.messages = service.NewMessageService(c.apiClient, c.cdnClient, c.contextTokens, c.middleware)
}

// UsePlugin registers a plugin and initializes it.
// The plugin's Initialize method is called synchronously.
func (c *Client) UsePlugin(ctx context.Context, p plugin.Plugin) error {
	if err := c.plugins.Register(p); err != nil {
		return err
	}
	return p.Initialize(ctx, c)
}

// UsePluginSimple registers a plugin without a context.
// Deprecated: Use UsePlugin(ctx, plugin) instead.
func (c *Client) UsePluginSimple(p plugin.Plugin) error {
	return c.UsePlugin(context.Background(), p)
}

// --- Context token accessors ---

// GetContextToken returns the context token for a user.
func (c *Client) GetContextToken(accountID, userID string) string {
	return c.contextTokens.Get(accountID, userID)
}

// SetContextToken sets the context token for a user.
func (c *Client) SetContextToken(accountID, userID, token string) {
	c.contextTokens.Set(accountID, userID, token)
}

// --- Service accessors (optional, for advanced users) ---

// Messages returns the message service.
func (c *Client) Messages() service.MessageService { return c.messages }

// Media returns the media service.
func (c *Client) Media() service.MediaService { return c.media }

// Auth returns the auth service.
func (c *Client) Auth() service.AuthService { return c.auth }

// Session returns the session service.
func (c *Client) Session() service.SessionService { return c.session }

// Events returns the event dispatcher for subscribing to SDK events.
//
// Example:
//
//	client.Events().Subscribe(event.EventTypeMessage, func(ctx context.Context, e *event.Event) error {
//	    msg := e.Data.(*ilink.Message)
//	    log.Printf("收到消息: %v", msg)
//	    return nil
//	})
func (c *Client) Events() *event.Dispatcher { return c.events }

// --- Convenience event subscription methods ---

// OnMessage registers a handler for message events.
// This is a convenience method equivalent to Events().Subscribe(event.EventTypeMessage, handler).
func (c *Client) OnMessage(handler event.Handler) {
	c.events.Subscribe(event.EventTypeMessage, handler)
}

// OnError registers a handler for error events.
// This is a convenience method equivalent to Events().Subscribe(event.EventTypeError, handler).
func (c *Client) OnError(handler event.Handler) {
	c.events.Subscribe(event.EventTypeError, handler)
}

// OnLogin registers a handler for login events.
// This is a convenience method equivalent to Events().Subscribe(event.EventTypeLogin, handler).
func (c *Client) OnLogin(handler event.Handler) {
	c.events.Subscribe(event.EventTypeLogin, handler)
}

// OnSessionExpired registers a handler for session expired events.
// This is a convenience method equivalent to Events().Subscribe(event.EventTypeSessionExpired, handler).
func (c *Client) OnSessionExpired(handler event.Handler) {
	c.events.Subscribe(event.EventTypeSessionExpired, handler)
}

// OnConnected registers a handler for connected events.
// This is a convenience method equivalent to Events().Subscribe(event.EventTypeConnected, handler).
func (c *Client) OnConnected(handler event.Handler) {
	c.events.Subscribe(event.EventTypeConnected, handler)
}

// OnDisconnected registers a handler for disconnected events.
// This is a convenience method equivalent to Events().Subscribe(event.EventTypeDisconnected, handler).
func (c *Client) OnDisconnected(handler event.Handler) {
	c.events.Subscribe(event.EventTypeDisconnected, handler)
}

// --- Utility methods ---

// Close stops the client and releases resources.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		close(c.stopChan)
		c.running = false
	}
	return nil
}

// Logger returns the configured logger.
func (c *Client) Logger() *slog.Logger {
	return c.config.Logger
}