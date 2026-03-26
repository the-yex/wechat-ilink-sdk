// Package plugin provides a plugin system for extending SDK functionality.
package plugin

import (
	"context"

	"github.com/the-yex/wechat-ilink-sdk/ilink"
	"github.com/the-yex/wechat-ilink-sdk/media"
)

// Plugin extends SDK functionality.
type Plugin interface {
	// Name returns the plugin name.
	Name() string

	// Initialize is called when the plugin is registered.
	Initialize(ctx context.Context, sdk SDK) error

	// OnMessage is called for each received message.
	// Return nil to allow other plugins/handlers to process.
	// Return error to stop processing.
	OnMessage(ctx context.Context, msg *ilink.Message) error

	// OnError is called when an error occurs.
	OnError(ctx context.Context, err error)
}

// SDK provides plugin access to SDK functionality.
type SDK interface {
	// SendMessage sends a message.
	SendMessage(ctx context.Context, req *ilink.SendMessageRequest) error
	// SendText sends a text message.
	SendText(ctx context.Context, toUserID, text string) error
	// UploadMedia uploads a media file.
	UploadMedia(ctx context.Context, req *media.UploadRequest) (*media.UploadResult, error)
	// DownloadMedia downloads and decrypts a media file.
	DownloadMedia(ctx context.Context, req *media.DownloadRequest) ([]byte, error)
	// Logout clears the stored token and triggers re-login.
	// After calling this, the user will need to scan QR code again.
	Logout(ctx context.Context) error
}