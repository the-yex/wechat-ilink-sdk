// Package service provides focused services for the WeChat iLink SDK.
//
// Each service handles a specific domain:
//   - MessageService: Message sending (text, image, typing)
//   - MediaService: CDN media upload/download
//   - AuthService: Authentication and token management
//   - SessionService: Session state queries
package service

import (
	"context"
	"time"

	"github.com/the-yex/wechat-ilink-sdk/ilink"
	"github.com/the-yex/wechat-ilink-sdk/login"
	"github.com/the-yex/wechat-ilink-sdk/media"
)

// APIClient is the interface for iLink API operations needed by MessageService.
type APIClient interface {
	SendMessage(ctx context.Context, req *ilink.SendMessageRequest) error
	SendTyping(ctx context.Context, req *ilink.SendTypingRequest) error
	GetConfig(ctx context.Context, req *ilink.GetConfigRequest) (*ilink.GetConfigResponse, error)
}

// CDNClient is the interface for CDN operations needed by MessageService.
type CDNClient interface {
	Upload(ctx context.Context, req *media.UploadRequest) (*media.UploadResult, error)
	Download(ctx context.Context, req *media.DownloadRequest) ([]byte, error)
}

// MessageService handles message operations.
type MessageService interface {
	// SendMessage sends a message with the given request.
	SendMessage(ctx context.Context, req *ilink.SendMessageRequest) error

	// SendText sends a text message to a user.
	SendText(ctx context.Context, toUserID, text string) error

	// SendImage sends an image message to a user.
	// The image data is automatically uploaded to CDN.
	SendImage(ctx context.Context, toUserID string, imageData []byte) error

	// SendTyping sends a typing indicator to a user.
	SendTyping(ctx context.Context, toUserID string, typing bool) error
}

// MediaService handles media operations.
type MediaService interface {
	// Upload uploads a media file to CDN.
	Upload(ctx context.Context, req *media.UploadRequest) (*media.UploadResult, error)

	// Download downloads and decrypts a media file from CDN.
	Download(ctx context.Context, req *media.DownloadRequest) ([]byte, error)
}

// AuthService handles authentication operations.
type AuthService interface {
	// Login performs QR code login.
	Login(ctx context.Context, displayCallback login.QRCodeCallback) (*ilink.LoginResult, error)

	// SetToken sets the authentication token.
	SetToken(token, baseURL string)

	// LoadToken loads a stored token for an account.
	LoadToken(accountID string) error

	// ListAccounts lists all stored account IDs.
	ListAccounts() ([]string, error)
}

// SessionService handles session state queries.
type SessionService interface {
	// IsPaused returns true if the session is paused.
	IsPaused() bool

	// RemainingPause returns the remaining pause duration.
	RemainingPause() time.Duration
}

// ContextTokenService handles context token management.
type ContextTokenService interface {
	// Get retrieves a context token.
	Get(accountID, userID string) string

	// Set stores a context token.
	Set(accountID, userID, token string)

	// Delete removes a context token.
	Delete(accountID, userID string)
}