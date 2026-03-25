// Package types provides type definitions for the iLink SDK.
package types

// BaseInfo contains common request metadata attached to every CGI request.
type BaseInfo struct {
	ChannelVersion string `json:"channel_version,omitempty"`
}

// MessageType constants.
type MessageType int

const (
	MessageTypeNone MessageType = iota
	MessageTypeUser             // User message
	MessageTypeBot              // Bot message
)

// MessageItemType constants.
type MessageItemType int

const (
	MessageItemTypeNone MessageItemType = iota
	MessageItemTypeText
	MessageItemTypeImage
	MessageItemTypeVoice
	MessageItemTypeFile
	MessageItemTypeVideo
)

// MessageState constants.
type MessageState int

const (
	MessageStateNew MessageState = iota
	MessageStateGenerating
	MessageStateFinish
)

// UploadMediaType constants for CDN upload.
type UploadMediaType int

const (
	UploadMediaTypeImage UploadMediaType = iota + 1
	UploadMediaTypeVideo
	UploadMediaTypeFile
	UploadMediaTypeVoice
)

// TypingStatus constants.
type TypingStatus int

const (
	TypingStatusTyping TypingStatus = iota + 1
	TypingStatusCancel
)

// LoginStatus represents QR code login status.
// API returns string values: "wait", "scaned", "confirmed", "expired"
type LoginStatus string

const (
	LoginStatusWaiting  LoginStatus = "wait"      // Waiting for scan
	LoginStatusScanned  LoginStatus = "scaned"    // QR code scanned
	LoginStatusConfirmed LoginStatus = "confirmed" // Login confirmed
	LoginStatusExpired  LoginStatus = "expired"   // QR code expired
	LoginStatusCanceled LoginStatus = "canceled"  // Login canceled (not used by API)
)