package ilinksdk

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/the-yex/wechat-ilink-sdk/event"
	"github.com/the-yex/wechat-ilink-sdk/ilink"
	"github.com/the-yex/wechat-ilink-sdk/internal/contextmgr"
	"github.com/the-yex/wechat-ilink-sdk/internal/service"
	"github.com/the-yex/wechat-ilink-sdk/login"
	"github.com/the-yex/wechat-ilink-sdk/media"
	"github.com/the-yex/wechat-ilink-sdk/middleware"
	"github.com/the-yex/wechat-ilink-sdk/plugin"
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
	handlers   *messageHandlers

	// Polling state
	mu       sync.Mutex
	running  bool
	stopChan chan struct{}

	// Current user info (cached from login/token load)
	currentUser *ilink.LoginResult
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
		handlers:      &messageHandlers{},
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

	// Auto-load token - priority: TokenProvider > TokenStore
	// This allows seamless re-authentication without QR code scan
	if cfg.TokenProvider != nil {
		// User provides their own token loading logic
		tokenInfo, err := cfg.TokenProvider(context.Background())
		if err == nil && tokenInfo != nil && tokenInfo.Token != "" {
			client.onTokenUpdate(tokenInfo.Token, tokenInfo.BaseURL, login.DefaultAccountID, tokenInfo.UserID)
			cfg.Logger.Debug("loaded token from provider")
		}
	} else if cfg.TokenStore != nil {
		// Default: load from TokenStore
		tokenInfo, err := tokenStore.Load(login.DefaultAccountID)
		if err == nil && tokenInfo != nil && tokenInfo.Token != "" {
			client.onTokenUpdate(tokenInfo.Token, tokenInfo.BaseURL, login.DefaultAccountID, tokenInfo.UserID)
			cfg.Logger.Debug("loaded stored token")
		}
	}

	// Set default OnLogin callback if not provided - display QR code in terminal
	if cfg.OnLogin == nil {
		cfg.OnLogin = func(ctx context.Context, qr *login.QRCode) error {
			login.PrintQRCodeWithTerm(qr)
			return nil
		}
	}

	// Set default OnSessionExpired callback if not provided - auto re-login
	if cfg.OnSessionExpired == nil {
		cfg.OnSessionExpired = func(ctx context.Context) (*ilink.LoginResult, error) {
			cfg.Logger.Info("session expired, please re-scan QR code to login")
			return client.Login(ctx, cfg.OnLogin)
		}
	}

	return client, nil
}

// onTokenUpdate handles token updates from AuthService.
// It recreates the apiClient and cdnClient with the new token.
func (c *Client) onTokenUpdate(token, baseURL, accountID, userID string) {
	if baseURL != "" {
		c.config.BaseURL = baseURL
	}
	c.config.Token = token // Update config token so IsLoggedIn() works

	// Update current user info
	c.currentUser = &ilink.LoginResult{
		Token:     token,
		AccountID: accountID,
		UserID:    userID,
		BaseURL:   baseURL,
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

// clearToken clears the stored token.
// If using TokenProvider, calls OnTokenInvalid callback.
// If using TokenStore, deletes from store.
func (c *Client) clearToken(ctx context.Context) {
	if c.config.TokenProvider != nil && c.config.OnTokenInvalid != nil {
		// User is managing tokens themselves
		c.config.OnTokenInvalid(ctx)
	} else if c.tokenStore != nil {
		_ = c.tokenStore.Delete(login.DefaultAccountID)
	}
	// Reset login state
	c.config.Token = ""
	c.currentUser = nil
}

// Run starts the message polling loop and processes messages with the given handler.
// This is a blocking call. Use context cancellation to stop.
//
// If not already logged in and OnLogin callback is set, Run will automatically
// trigger the login flow before starting the message loop.
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
	}()

	// Auto-login if not already logged in
	if !c.IsLoggedIn() {
		c.config.Logger.Info("auto-login: not logged in, triggering login flow")
		if _, err := c.Login(ctx, c.config.OnLogin); err != nil {
			return fmt.Errorf("auto-login failed: %w", err)
		}
	}

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

		// Check if session is paused (session timeout)
		if c.apiClient.IsPaused() {
			// Dispatch session expired event
			c.events.Dispatch(ctx, &event.Event{
				Type:    event.EventTypeSessionExpired,
				Context: ctx,
			})
			c.config.Logger.Warn("session expired, triggering re-login callback")

			// Clear stored token
			c.clearToken(ctx)

			// Call the session expired callback if set
			if c.config.OnSessionExpired != nil {
				result, err := c.config.OnSessionExpired(ctx)
				if err != nil {
					c.config.Logger.Error("re-login failed", "error", err)
					return fmt.Errorf("re-login failed: %w", err)
				}
				if result == nil {
					// Callback returned nil, stop the loop
					return nil
				}
				// Re-login successful, reset session guard and continue
				c.apiClient.ResetSession()
				c.config.Logger.Info("re-login successful, continuing message loop")
				continue
			}

			// No callback, return error
			return fmt.Errorf("session expired, please re-login")
		}

		// Long poll for messages
		resp, err := c.apiClient.GetUpdates(ctx, &ilink.GetUpdatesRequest{
			GetUpdatesBuf: getUpdatesBuf,
		})
		if err != nil {
			// Check if it's an authentication error (token expired)
			if isAuthError(err) {
				c.config.Logger.Warn("token expired, triggering re-login callback")

				// Clear stored token
				c.clearToken(ctx)

				// Call the session expired callback if set
				if c.config.OnSessionExpired != nil {
					result, callbackErr := c.config.OnSessionExpired(ctx)
					if callbackErr != nil {
						c.config.Logger.Error("re-login failed", "error", callbackErr)
						return fmt.Errorf("re-login failed: %w", callbackErr)
					}
					if result == nil {
						// Callback returned nil, stop the loop
						return nil
					}
					// Re-login successful, reset session guard and continue
					c.apiClient.ResetSession()
					c.config.Logger.Info("re-login successful, continuing message loop")
					continue
				}
			}
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
			// Store context token (use empty accountID for simple lookup)
			if msg.ContextToken != "" && msg.FromUserID != "" {
				c.contextTokens.Set("", msg.FromUserID, msg.ContextToken)
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
			} else if c.handlers.hasAnyHandler() {
				// Use type-specific handlers if registered
				if err := c.handlers.buildHandler()(ctx, msg); err != nil {
					c.plugins.OnError(ctx, err)
					c.config.Logger.Error("handler error", "error", err)
				}
			}
		}
	}
}

// MessageHandler handles received messages.
type MessageHandler func(ctx context.Context, msg *ilink.Message) error

// --- Type-specific message handlers ---

// OnText registers a handler for text messages.
// If set, Run() will use it automatically when no explicit handler is passed.
func (c *Client) OnText(handler TextHandler) {
	c.handlers.textHandler = handler
}

// OnImage registers a handler for image messages.
func (c *Client) OnImage(handler ImageHandler) {
	c.handlers.imageHandler = handler
}

// OnVideo registers a handler for video messages.
func (c *Client) OnVideo(handler VideoHandler) {
	c.handlers.videoHandler = handler
}

// OnVoice registers a handler for voice messages.
func (c *Client) OnVoice(handler VoiceHandler) {
	c.handlers.voiceHandler = handler
}

// OnFile registers a handler for file messages.
func (c *Client) OnFile(handler FileHandler) {
	c.handlers.fileHandler = handler
}

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

// SendVideo sends a video message.
func (c *Client) SendVideo(ctx context.Context, toUserID string, videoData []byte) error {
	return c.messages.SendVideo(ctx, toUserID, videoData)
}

// SendVoice sends a voice message.
// voiceItem should contain playtime, encode_type, bits_per_sample, sample_rate from the original message.
func (c *Client) SendVoice(ctx context.Context, toUserID string, voiceData []byte, voiceItem *ilink.VoiceItem) error {
	return c.messages.SendVoice(ctx, toUserID, voiceData, voiceItem)
}

// SendFile sends a file message.
func (c *Client) SendFile(ctx context.Context, toUserID, fileName string, fileData []byte) error {
	return c.messages.SendFile(ctx, toUserID, fileName, fileData)
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
// If a valid token is already stored, it returns the cached login result without QR code scan.
// The displayCallback is called with context and the QR code for display (only if scan is needed).
func (c *Client) Login(ctx context.Context, displayCallback login.QRCodeCallback) (*ilink.LoginResult, error) {
	// If already logged in with a valid token, verify it's still valid
	if c.IsLoggedIn() && c.currentUser != nil {
		c.config.Logger.Debug("verifying stored token")

		// Verify token by calling GetConfig API
		resp, err := c.apiClient.GetConfig(ctx, &ilink.GetConfigRequest{
			ILinkUserID:  c.currentUser.UserID,
			ContextToken: c.currentUser.Token,
			BaseInfo: ilink.BaseInfo{
				ChannelVersion: Version,
			},
		})
		if err == nil && resp != nil && resp.ErrCode == 0 {
			c.config.Logger.Debug("token is valid, skipping QR code scan")
			return c.currentUser, nil
		}

		// Token invalid, log the reason
		c.config.Logger.Warn("stored token is invalid, will perform QR code login")
	}

	// Clear invalid token
	c.clearToken(ctx)

	result, err := c.auth.Login(ctx, displayCallback)
	if err != nil {
		return nil, err
	}

	// Call OnLoginSuccess callback if set (for user to save login info)
	if c.config.OnLoginSuccess != nil {
		if err := c.config.OnLoginSuccess(ctx, result); err != nil {
			c.config.Logger.Warn("OnLoginSuccess callback failed", "error", err)
		}
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
func (c *Client) SetToken(token, baseURL, accountID, userID string) {
	c.auth.SetToken(token, baseURL, accountID, userID)
}

// LoadToken loads a stored token for an account.
func (c *Client) LoadToken(accountID string) error {
	return c.auth.LoadToken(accountID)
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

// SetOnSessionExpired sets the callback for session expiration.
// This allows setting the callback after client creation, which is useful
// when the callback needs to reference the client itself.
func (c *Client) SetOnSessionExpired(callback SessionExpiredCallback) {
	c.config.OnSessionExpired = callback
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

// IsLoggedIn returns true if a token is configured.
func (c *Client) IsLoggedIn() bool {
	return c.config.Token != ""
}

// CurrentUser returns the current logged-in user info.
func (c *Client) CurrentUser() *ilink.LoginResult {
	return c.currentUser
}

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

// isAuthError checks if an error is related to authentication failure.
func isAuthError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "401") ||
		strings.Contains(errStr, "unauthorized") ||
		strings.Contains(errStr, "invalid token") ||
		strings.Contains(errStr, "token expired") ||
		strings.Contains(errStr, "session timeout")
}
