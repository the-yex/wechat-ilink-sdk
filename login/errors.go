package login

import "errors"

var (
	// ErrQRCodeExpired indicates the QR code expired before login completed.
	ErrQRCodeExpired = errors.New("qr code expired")

	// ErrLoginCanceled indicates the user canceled login from WeChat.
	ErrLoginCanceled = errors.New("login canceled")

	// ErrUnknownStatus indicates the server returned an unexpected login status.
	ErrUnknownStatus = errors.New("unknown login status")
)
