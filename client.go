package ilinksdk

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
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
	mu        sync.Mutex
	running   bool
	stopChan  chan struct{}
	closeOnce sync.Once

	// Login state (atomic for thread-safe access without locks)
	closed      atomic.Bool
	loggedIn    atomic.Bool
	currentUser atomic.Pointer[ilink.LoginResult]
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
	apiClient.SetVersion(Version)

	// Create CDN client
	cdnClient := media.NewClient(cfg.CDNBaseURL, apiClient)

	effectiveMiddleware := buildMiddleware(cfg)

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
		middleware:    effectiveMiddleware,
		events:        event.NewDispatcher(),
		handlers:      &messageHandlers{},
		stopChan:      make(chan struct{}),
	}

	// Initialize services
	client.messages = service.NewMessageService(apiClient, cdnClient, contextTokens, effectiveMiddleware)
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
// It updates the token without recreating clients to preserve connection pools.
func (c *Client) onTokenUpdate(token, baseURL, accountID, userID string) {
	if baseURL != "" {
		c.config.BaseURL = baseURL
	}

	// Update current user info atomically
	c.currentUser.Store(&ilink.LoginResult{
		Token:     token,
		AccountID: accountID,
		UserID:    userID,
		BaseURL:   baseURL,
	})
	c.loggedIn.Store(true)

	// Update token on existing clients (preserves connection pool)
	c.apiClient.SetToken(token)
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

	// Drop in-memory state so a new login starts from a clean session.
	c.contextTokens.Clear()

	// Reset login state atomically
	c.loggedIn.Store(false)
	c.currentUser.Store(nil)
	c.apiClient.SetToken("")
}

// Run starts the message polling loop and processes messages with the given handler.
// This is a blocking call. Use context cancellation to stop.
//
// If not already logged in and OnLogin callback is set, Run will automatically
// trigger the login flow before starting the message loop.
func (c *Client) Run(ctx context.Context, handler MessageHandler) error {
	if err := c.ensureOpen("run"); err != nil {
		return err
	}

	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return fmt.Errorf("client is already running")
	}
	c.running = true
	c.mu.Unlock()
	connected := false
	defer func() {
		c.mu.Lock()
		c.running = false
		c.mu.Unlock()
		if connected {
			c.events.Dispatch(context.Background(), &event.Event{
				Type:    event.EventTypeDisconnected,
				Context: context.Background(),
			})
		}
	}()

	// Auto-login if not already logged in
	if !c.IsLoggedIn() {
		c.config.Logger.Info("auto-login: not logged in, triggering login flow")
		if _, err := c.Login(ctx, c.config.OnLogin); err != nil {
			return wrapError("run", fmt.Errorf("auto-login failed: %w", err), nil)
		}
	}

	// Initialize plugins
	if err := c.plugins.Initialize(ctx); err != nil {
		return wrapError("run", fmt.Errorf("initialize plugins: %w", err), nil)
	}

	// Dispatch connected event
	c.events.Dispatch(ctx, &event.Event{
		Type:    event.EventTypeConnected,
		Context: ctx,
	})
	connected = true

	// Get updates buffer
	var getUpdatesBuf string
	consecutivePollErrors := 0

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
					return wrapError("run", fmt.Errorf("re-login failed: %w", err), nil)
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
			return wrapError("run", ErrSessionExpired, nil)
		}

		// Long poll for messages
		resp, err := c.apiClient.GetUpdates(ctx, &ilink.GetUpdatesRequest{
			GetUpdatesBuf: getUpdatesBuf,
		})
		if err != nil {
			err = wrapError("get updates", err, nil)

			// Ignore context cancellation (normal shutdown)
			if errors.Is(err, context.Canceled) {
				return ctx.Err()
			}

			// Check if it's an authentication error (token expired)
			if IsAuthenticationError(err) {
				c.config.Logger.Warn("token expired, triggering re-login callback")

				// Clear stored token
				c.clearToken(ctx)

				// Call the session expired callback if set
				if c.config.OnSessionExpired != nil {
					result, callbackErr := c.config.OnSessionExpired(ctx)
					if callbackErr != nil {
						c.config.Logger.Error("re-login failed", "error", callbackErr)
						return wrapError("run", fmt.Errorf("re-login failed: %w", callbackErr), nil)
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

			consecutivePollErrors++
			if backoffErr := c.waitPollErrorBackoff(ctx, consecutivePollErrors, err); backoffErr != nil {
				return backoffErr
			}
			continue
		}

		consecutivePollErrors = 0

		// Update buffer for next request
		if resp.GetUpdatesBuf != "" {
			getUpdatesBuf = resp.GetUpdatesBuf
		}

		// Process messages
		for _, msg := range resp.Messages {
			// Store context token
			if msg.ContextToken != "" && msg.FromUserID != "" {
				c.contextTokens.Set(msg.FromUserID, msg.ContextToken)
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

// OnMessage registers a general message handler.
// If set, Run() will use it automatically when no explicit handler is passed.
// This is useful when you want to handle all message types in one place.
func (c *Client) OnMessage(handler MessageHandler) {
	c.handlers.messageHandler = handler
	c.handlers.cachedHandler = nil // Invalidate cache
}

// OnText registers a handler for text messages.
// If set, Run() will use it automatically when no explicit handler is passed.
func (c *Client) OnText(handler TextHandler) {
	c.handlers.textHandler = handler
	c.handlers.cachedHandler = nil // Invalidate cache
}

// OnImage registers a handler for image messages.
func (c *Client) OnImage(handler ImageHandler) {
	c.handlers.imageHandler = handler
	c.handlers.cachedHandler = nil // Invalidate cache
}

// OnVideo registers a handler for video messages.
func (c *Client) OnVideo(handler VideoHandler) {
	c.handlers.videoHandler = handler
	c.handlers.cachedHandler = nil // Invalidate cache
}

// OnVoice registers a handler for voice messages.
func (c *Client) OnVoice(handler VoiceHandler) {
	c.handlers.voiceHandler = handler
	c.handlers.cachedHandler = nil // Invalidate cache
}

// OnFile registers a handler for file messages.
func (c *Client) OnFile(handler FileHandler) {
	c.handlers.fileHandler = handler
	c.handlers.cachedHandler = nil // Invalidate cache
}

// --- MessageService delegation ---

// SendMessage sends a message.
func (c *Client) SendMessage(ctx context.Context, req *ilink.SendMessageRequest) error {
	if err := c.ensureOpen("send message"); err != nil {
		return err
	}
	return wrapError("send message", c.messages.SendMessage(ctx, req), nil)
}

// SendText sends a text message.
func (c *Client) SendText(ctx context.Context, toUserID, text string) error {
	if err := c.ensureOpen("send text"); err != nil {
		return err
	}
	return wrapError("send text", c.messages.SendText(ctx, toUserID, text), nil)
}

// SendImage sends an image message.
func (c *Client) SendImage(ctx context.Context, toUserID string, imageData []byte) error {
	if err := c.ensureOpen("send image"); err != nil {
		return err
	}
	return wrapError("send image", c.messages.SendImage(ctx, toUserID, imageData), ErrUploadFailed)
}

// SendVideo sends a video message.
func (c *Client) SendVideo(ctx context.Context, toUserID string, videoData []byte) error {
	if err := c.ensureOpen("send video"); err != nil {
		return err
	}
	return wrapError("send video", c.messages.SendVideo(ctx, toUserID, videoData), ErrUploadFailed)
}

// SendVoice sends a voice message.
// voiceItem should contain playtime, encode_type, bits_per_sample, sample_rate from the original message.
func (c *Client) SendVoice(ctx context.Context, toUserID string, voiceData []byte, voiceItem *ilink.VoiceItem) error {
	if err := c.ensureOpen("send voice"); err != nil {
		return err
	}
	return wrapError("send voice", c.messages.SendVoice(ctx, toUserID, voiceData, voiceItem), ErrUploadFailed)
}

// SendFile sends a file message.
func (c *Client) SendFile(ctx context.Context, toUserID, fileName string, fileData []byte) error {
	if err := c.ensureOpen("send file"); err != nil {
		return err
	}
	return wrapError("send file", c.messages.SendFile(ctx, toUserID, fileName, fileData), ErrUploadFailed)
}

// SendTyping sends a typing indicator.
func (c *Client) SendTyping(ctx context.Context, toUserID string, typing bool) error {
	if err := c.ensureOpen("send typing"); err != nil {
		return err
	}
	return wrapError("send typing", c.messages.SendTyping(ctx, toUserID, typing), nil)
}

// --- MediaService delegation ---

// UploadMedia uploads a media file to CDN.
func (c *Client) UploadMedia(ctx context.Context, req *media.UploadRequest) (*media.UploadResult, error) {
	if err := c.ensureOpen("upload media"); err != nil {
		return nil, err
	}
	result, err := c.media.Upload(ctx, req)
	return result, wrapError("upload media", err, ErrUploadFailed)
}

// DownloadMedia downloads and decrypts a media file from CDN.
func (c *Client) DownloadMedia(ctx context.Context, req *media.DownloadRequest) ([]byte, error) {
	if err := c.ensureOpen("download media"); err != nil {
		return nil, err
	}
	data, err := c.media.Download(ctx, req)
	return data, wrapError("download media", err, ErrDownloadFailed)
}

// --- AuthService delegation ---

// Login performs QR code login and returns the login result.
// If a valid token is already stored, it returns the cached login result without QR code scan.
// The displayCallback is called with context and the QR code for display (only if scan is needed).
func (c *Client) Login(ctx context.Context, displayCallback login.QRCodeCallback) (*ilink.LoginResult, error) {
	if err := c.ensureOpen("login"); err != nil {
		return nil, err
	}

	// If already logged in with a valid token, verify it's still valid
	user := c.currentUser.Load()
	if c.IsLoggedIn() && user != nil {
		c.config.Logger.Debug("verifying stored token")

		// Verify token by calling GetConfig API
		resp, err := c.apiClient.GetConfig(ctx, &ilink.GetConfigRequest{
			ILinkUserID:  user.UserID,
			ContextToken: user.Token,
			BaseInfo: ilink.BaseInfo{
				ChannelVersion: Version,
			},
		})
		if err == nil && resp != nil && resp.ErrCode == 0 {
			c.config.Logger.Debug("token is valid, skipping QR code scan")
			return user, nil
		}

		// Token invalid, log the reason
		c.config.Logger.Warn("stored token is invalid, will perform QR code login")
	}

	// Clear invalid token
	c.clearToken(ctx)

	result, err := c.auth.Login(ctx, displayCallback)
	if err != nil {
		return nil, wrapError("login", err, nil)
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
	if c.closed.Load() {
		return
	}
	c.auth.SetToken(token, baseURL, accountID, userID)
}

// LoadToken loads a stored token for an account.
func (c *Client) LoadToken(accountID string) error {
	if err := c.ensureOpen("load token"); err != nil {
		return err
	}
	return wrapError("load token", c.auth.LoadToken(accountID), nil)
}

// Logout clears the stored token and triggers re-login.
// After calling this, the SDK will pause the current session and
// trigger the OnSessionExpired callback, which by default shows a QR code for re-login.
func (c *Client) Logout(ctx context.Context) error {
	if err := c.ensureOpen("logout"); err != nil {
		return err
	}

	// Clear stored token
	c.clearToken(ctx)

	// Reset session guard first
	c.apiClient.ResetSession()

	// Pause session to trigger re-login flow in Run()
	c.apiClient.PauseSession()

	return nil
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
	if c.closed.Load() {
		return
	}
	c.config.Middleware = append(c.config.Middleware, m...)
	c.middleware = buildMiddleware(c.config)
	// Update MessageService with new middleware
	c.messages = service.NewMessageService(c.apiClient, c.cdnClient, c.contextTokens, c.middleware)
}

// UsePlugin registers a plugin and initializes it.
// The plugin's Initialize method is called synchronously.
func (c *Client) UsePlugin(p plugin.Plugin) error {
	if err := c.ensureOpen("register plugin"); err != nil {
		return err
	}
	if err := c.plugins.Register(p); err != nil {
		return wrapError("register plugin", err, nil)
	}
	return wrapError("initialize plugin", c.plugins.InitializeOne(context.Background(), p), nil)
}

// --- Context token accessors ---

// GetContextToken returns the context token for a user.
func (c *Client) GetContextToken(userID string) string {
	return c.contextTokens.Get(userID)
}

// SetContextToken sets the context token for a user.
func (c *Client) SetContextToken(userID, token string) {
	if c.closed.Load() {
		return
	}
	c.contextTokens.Set(userID, token)
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
	return c.loggedIn.Load()
}

// CurrentUser returns the current logged-in user info.
func (c *Client) CurrentUser() *ilink.LoginResult {
	return c.currentUser.Load()
}

// Events returns the event dispatcher for subscribing to SDK events.
//
// Example:
//
//	client.Events().Subscribe(event.EventTypeLogin, func(ctx context.Context, e *event.Event) error {
//	    result := e.Data.(*ilink.LoginResult)
//	    log.Printf("登录成功: %s", result.UserID)
//	    return nil
//	})
func (c *Client) Events() *event.Dispatcher { return c.events }

// --- Utility methods ---

// Close stops the client and releases resources.
// After Close returns, active operations such as Run, Login, Send*, UploadMedia,
// DownloadMedia, Logout, and plugin registration will return ErrClientClosed.
// It is safe to call Close multiple times.
func (c *Client) Close() error {
	c.closeOnce.Do(func() {
		c.closed.Store(true)
		c.mu.Lock()
		defer c.mu.Unlock()

		if c.running {
			close(c.stopChan)
			c.running = false
		}
	})
	return nil
}

// Logger returns the configured logger.
func (c *Client) Logger() *slog.Logger {
	return c.config.Logger
}

func (c *Client) ensureOpen(op string) error {
	if !c.closed.Load() {
		return nil
	}
	return &Error{
		Op:   op,
		Kind: ErrClientClosed,
		Err:  ErrClientClosed,
	}
}
