package ilinksdk

import (
	"errors"
)

// Sentinel errors for the SDK.
var (
	// ErrInvalidConfig indicates invalid client configuration.
	ErrInvalidConfig = errors.New("invalid configuration")

	// ErrTokenRequired indicates token is required but not provided.
	ErrTokenRequired = errors.New("token is required")

	// ErrSessionExpired indicates the bot session has expired (errcode -14).
	ErrSessionExpired = errors.New("session expired")

	// ErrSessionPaused indicates API calls are paused due to session expiry.
	ErrSessionPaused = errors.New("session is paused, please wait")

	// ErrContextTokenRequired indicates context token is required for sending messages.
	ErrContextTokenRequired = errors.New("context token is required")

	// ErrInvalidMediaType indicates an invalid media type.
	ErrInvalidMediaType = errors.New("invalid media type")

	// ErrUploadFailed indicates media upload to CDN failed.
	ErrUploadFailed = errors.New("media upload failed")

	// ErrDownloadFailed indicates media download from CDN failed.
	ErrDownloadFailed = errors.New("media download failed")

	// ErrEncryptionFailed indicates AES encryption failed.
	ErrEncryptionFailed = errors.New("encryption failed")

	// ErrDecryptionFailed indicates AES decryption failed.
	ErrDecryptionFailed = errors.New("decryption failed")

	// ErrInvalidAESKey indicates invalid AES key length.
	ErrInvalidAESKey = errors.New("invalid AES key: must be 16 bytes")

	// ErrInvalidCiphertext indicates invalid ciphertext length.
	ErrInvalidCiphertext = errors.New("invalid ciphertext: must be multiple of 16 bytes")

	// ErrPluginAlreadyRegistered indicates a plugin with the same name already exists.
	ErrPluginAlreadyRegistered = errors.New("plugin already registered")

	// ErrMessageEmpty indicates the message has no content.
	ErrMessageEmpty = errors.New("message is empty")

	// ErrNoTokenStore indicates no token store is configured.
	ErrNoTokenStore = errors.New("no token store configured")

	// ErrTokenNotFound indicates token not found for the specified account.
	ErrTokenNotFound = errors.New("token not found for account")

	// ErrQRCodeExpired indicates QR code has expired.
	ErrQRCodeExpired = errors.New("QR code expired")

	// ErrLoginCanceled indicates login was canceled by user.
	ErrLoginCanceled = errors.New("login canceled by user")
)