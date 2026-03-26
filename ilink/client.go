package ilink

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
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
	http    *http.Client // Shared HTTP client for normal requests
	httpLP  *http.Client // HTTP client for long-polling
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

// SetToken updates the authentication token without recreating the client.
// This preserves the HTTP connection pool and avoids unnecessary allocations.
func (c *Client) SetToken(token string) {
	c.config.Token = token
}

// buildBaseInfo creates the base_info for requests.
func (c *Client) buildBaseInfo() BaseInfo {
	return BaseInfo{ChannelVersion: c.version}
}

// randomWechatUin generates a random X-WECHAT-UIN header value.
func randomWechatUin() string {
	var b [4]byte
	rand.Read(b[:])
	u := binary.BigEndian.Uint32(b[:])
	// Use a small buffer to avoid heap allocation
	var buf [20]byte // uint32 max is 4294967295 (10 digits)
	n := bytesconv.AppendUint(buf[:0], u)
	return base64.StdEncoding.EncodeToString(n)
}

// bytesconv provides efficient byte conversion utilities.
var bytesconv = newBytesConv()

type bytesConv struct{}

func newBytesConv() *bytesConv { return &bytesConv{} }

// AppendUint appends the decimal representation of u to buf and returns the extended buffer.
func (*bytesConv) AppendUint(buf []byte, u uint32) []byte {
	if u == 0 {
		return append(buf, '0')
	}
	// Build digits in reverse order
	var digits [10]byte
	n := 0
	for u > 0 {
		digits[n] = byte('0' + u%10)
		u /= 10
		n++
	}
	// Reverse into buf
	for i := n - 1; i >= 0; i-- {
		buf = append(buf, digits[i])
	}
	return buf
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

	// Check HTTP status
	if resp.StatusCode >= 400 {
		return &APIError{
			Code:    resp.StatusCode,
			Message: "response status: " + resp.Status,
		}
	}

	// Decode response (only if respBody is provided)
	if respBody != nil {
		if err = json.NewDecoder(resp.Body).Decode(respBody); err != nil {
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
	req.BaseInfo = c.buildBaseInfo()
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
	req.BaseInfo = c.buildBaseInfo()
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

// ResetSession resets the session pause state.
// This should be called after successful re-login.
func (c *Client) ResetSession() {
	c.session.Reset()
}

// PauseSession pauses the session to trigger re-login flow.
// This is used by Logout to force the Run loop to trigger OnSessionExpired callback.
func (c *Client) PauseSession() {
	c.session.Pause()
}

// GetBotQRCode retrieves a QR code for bot login.
func (c *Client) GetBotQRCode(ctx context.Context, req *GetBotQRCodeRequest) (*GetBotQRCodeResponse, error) {
	// Build URL with query parameters
	baseURL := c.config.BaseURL + "ilink/bot/get_bot_qrcode"
	botType := req.BotType
	if botType == "" {
		botType = "3" // Default bot type
	}
	urlStr := fmt.Sprintf("%s?bot_type=%s", baseURL, url.QueryEscape(botType))

	// Build headers
	headers := c.buildHeaders(0)
	routeTag := loadRouteTag()
	if routeTag != "" {
		headers.Set("SKRouteTag", routeTag)
	}

	// Create request
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header = headers

	// Send request
	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get QR code failed: status=%d", resp.StatusCode)
	}
	// Parse response
	var result GetBotQRCodeResponse

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, nil
}

// loadRouteTag loads the SKRouteTag header value from environment.
// It checks ILINK_ROUTE_TAG environment variable first, then returns empty string.
func loadRouteTag() string {
	// Check environment variable
	if routeTag := os.Getenv("ILINK_ROUTE_TAG"); routeTag != "" {
		return routeTag
	}
	// Try legacy variable name
	if routeTag := os.Getenv("SK_ROUTE_TAG"); routeTag != "" {
		return routeTag
	}
	return ""
}

// GetQRCodeStatus checks the QR code scan status.
// It uses long polling (35 seconds) to wait for status changes.
func (c *Client) GetQRCodeStatus(ctx context.Context, req *GetQRCodeStatusRequest) (*GetQRCodeStatusResponse, error) {
	// Build URL with query parameters
	baseURL := c.config.BaseURL + "ilink/bot/get_qrcode_status"
	urlStr := fmt.Sprintf("%s?qrcode=%s", baseURL, url.QueryEscape(req.QRCode))

	// Build headers
	headers := c.buildHeaders(0)
	headers.Set("iLink-App-ClientVersion", "1")
	routeTag := loadRouteTag()
	if routeTag != "" {
		headers.Set("SKRouteTag", routeTag)
	}

	// Create request with long-poll timeout
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header = headers

	// Use long-poll client (35 second timeout)
	client := c.httpLP

	// Send request
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get status failed: status=%d, body=%s", resp.StatusCode, string(respData))
	}

	// Parse response
	var result GetQRCodeStatusResponse
	if err := json.Unmarshal(respData, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, nil
}
