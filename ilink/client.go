package ilink

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// DefaultTimeout is the default HTTP timeout.
	DefaultTimeout = 30 * time.Second
	// DefaultLongPollTimeout is the default long-poll timeout.
	DefaultLongPollTimeout = 35 * time.Second
	// SessionExpiredErrCode is returned when the bot session has expired.
	SessionExpiredErrCode = -14
	// UserAgent is the user agent string.
	UserAgent = "wechat-bot-sdk-go/1.0"
)

// ClientConfig holds the API client configuration.
type ClientConfig struct {
	BaseURL         string
	Token           string
	Timeout         time.Duration
	LongPollTimeout time.Duration
}

// Client handles all WeChat API communication.
type Client struct {
	config  ClientConfig
	http    *http.Client        // Shared HTTP client for normal requests
	httpLP  *http.Client        // HTTP client for long-polling
	session *SessionGuard
	version string
}

// NewClient creates a new API client.
func NewClient(cfg ClientConfig) *Client {
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultTimeout
	}
	if cfg.LongPollTimeout == 0 {
		cfg.LongPollTimeout = DefaultLongPollTimeout
	}
	// Ensure base URL ends with /
	if !strings.HasSuffix(cfg.BaseURL, "/") {
		cfg.BaseURL += "/"
	}

	// Create shared transport for connection pooling
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}

	return &Client{
		config:  cfg,
		http:    &http.Client{Timeout: cfg.Timeout, Transport: transport},
		httpLP:  &http.Client{Timeout: cfg.LongPollTimeout, Transport: transport},
		session: NewSessionGuard(),
		version: "1.0.0",
	}
}

// SetVersion sets the SDK version for base_info.
func (c *Client) SetVersion(v string) {
	c.version = v
}

// buildBaseInfo creates the base_info for requests.
func (c *Client) buildBaseInfo() BaseInfo {
	return BaseInfo{ChannelVersion: c.version}
}

// randomWechatUin generates a random X-WECHAT-UIN header value.
func randomWechatUin() string {
	b := make([]byte, 4)
	rand.Read(b)
	// Convert to uint32 then to string, then base64 encode
	u := uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%d", u)))
}

// buildHeaders creates the HTTP headers for a request.
func (c *Client) buildHeaders(bodyLen int) http.Header {
	h := make(http.Header)
	h.Set("Content-Type", "application/json")
	h.Set("AuthorizationType", "ilink_bot_token")
	h.Set("Content-Length", fmt.Sprintf("%d", bodyLen))
	h.Set("X-WECHAT-UIN", randomWechatUin())
	h.Set("User-Agent", UserAgent)
	if c.config.Token != "" {
		h.Set("Authorization", "Bearer "+c.config.Token)
	}
	return h
}

// doPost performs a POST request to the API.
func (c *Client) doPost(ctx context.Context, endpoint string, reqBody interface{}, respBody interface{}) error {
	// Check session guard
	if c.session.IsPaused() {
		return fmt.Errorf("session is paused, remaining: %v", c.session.RemainingPause())
	}

	// Encode request body
	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	// Build URL
	u, err := url.Parse(c.config.BaseURL + endpoint)
	if err != nil {
		return fmt.Errorf("parse url: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header = c.buildHeaders(len(body))

	// Select appropriate client (long-poll or normal)
	client := c.http
	if endpoint == "ilink/bot/getupdates" {
		client = c.httpLP
	}

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	// Check HTTP status
	if resp.StatusCode >= 400 {
		return &APIError{
			Code:    resp.StatusCode,
			Message: string(respData),
		}
	}

	// Decode response
	if respBody != nil && len(respData) > 0 {
		if err := json.Unmarshal(respData, respBody); err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}
	}

	return nil
}

// GetUpdates performs a long-poll getUpdates request.
func (c *Client) GetUpdates(ctx context.Context, req *GetUpdatesRequest) (*GetUpdatesResponse, error) {
	// Check session guard
	if c.session.IsPaused() {
		return nil, fmt.Errorf("session is paused")
	}

	// Set base info
	req.BaseInfo = c.buildBaseInfo()

	var resp GetUpdatesResponse
	if err := c.doPost(ctx, "ilink/bot/getupdates", req, &resp); err != nil {
		return nil, err
	}

	// Check for session expiry
	if resp.ErrCode == SessionExpiredErrCode {
		c.session.Pause()
		return nil, &APIError{Code: resp.ErrCode, Message: resp.ErrMsg}
	}

	return &resp, nil
}

// SendMessage sends a message downstream.
func (c *Client) SendMessage(ctx context.Context, req *SendMessageRequest) error {
	return c.doPost(ctx, "ilink/bot/sendmessage", req, nil)
}

// GetUploadURL retrieves a pre-signed CDN upload URL.
func (c *Client) GetUploadURL(ctx context.Context, req *GetUploadURLRequest) (*GetUploadURLResponse, error) {
	req.BaseInfo = c.buildBaseInfo()

	var resp GetUploadURLResponse
	if err := c.doPost(ctx, "ilink/bot/getuploadurl", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetConfig retrieves bot configuration including typing_ticket.
func (c *Client) GetConfig(ctx context.Context, req *GetConfigRequest) (*GetConfigResponse, error) {
	req.BaseInfo = c.buildBaseInfo()

	var resp GetConfigResponse
	if err := c.doPost(ctx, "ilink/bot/getconfig", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// SendTyping sends a typing indicator.
func (c *Client) SendTyping(ctx context.Context, req *SendTypingRequest) error {
	return c.doPost(ctx, "ilink/bot/sendtyping", req, nil)
}

// IsPaused returns true if the session is paused.
func (c *Client) IsPaused() bool {
	return c.session.IsPaused()
}

// RemainingPause returns the remaining pause duration.
func (c *Client) RemainingPause() time.Duration {
	return c.session.RemainingPause()
}

// GetBotQRCode retrieves a QR code for bot login.
func (c *Client) GetBotQRCode(ctx context.Context, req *GetBotQRCodeRequest) (*GetBotQRCodeResponse, error) {
	req.BaseInfo = c.buildBaseInfo()

	var resp GetBotQRCodeResponse
	if err := c.doPost(ctx, "ilink/bot/get_bot_qrcode", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetQRCodeStatus checks the QR code scan status.
func (c *Client) GetQRCodeStatus(ctx context.Context, req *GetQRCodeStatusRequest) (*GetQRCodeStatusResponse, error) {
	req.BaseInfo = c.buildBaseInfo()

	var resp GetQRCodeStatusResponse
	if err := c.doPost(ctx, "ilink/bot/get_qrcode_status", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}