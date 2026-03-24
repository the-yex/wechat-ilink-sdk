// Package ilink provides types and client for iLink Bot API.
package ilink

import (
	"github.com/the-yex/wechat-ilink-sdk/types"
)

// Type aliases for backward compatibility.
// These re-export types from the types package.
type (
	BaseInfo                = types.BaseInfo
	MessageType             = types.MessageType
	MessageItemType         = types.MessageItemType
	MessageState            = types.MessageState
	UploadMediaType         = types.UploadMediaType
	TypingStatus            = types.TypingStatus
	LoginStatus             = types.LoginStatus
	Message                 = types.Message
	MessageItem             = types.MessageItem
	TextItem                = types.TextItem
	CDNMedia                = types.CDNMedia
	ImageItem               = types.ImageItem
	VoiceItem               = types.VoiceItem
	FileItem                = types.FileItem
	VideoItem               = types.VideoItem
	RefMessage              = types.RefMessage
	GetUpdatesRequest       = types.GetUpdatesRequest
	GetUpdatesResponse      = types.GetUpdatesResponse
	SendMessageRequest      = types.SendMessageRequest
	SendMessageResponse     = types.SendMessageResponse
	GetUploadURLRequest     = types.GetUploadURLRequest
	GetUploadURLResponse    = types.GetUploadURLResponse
	SendTypingRequest       = types.SendTypingRequest
	SendTypingResponse      = types.SendTypingResponse
	GetConfigRequest        = types.GetConfigRequest
	GetConfigResponse       = types.GetConfigResponse
	LoginResult             = types.LoginResult
	GetBotQRCodeRequest     = types.GetBotQRCodeRequest
	GetBotQRCodeResponse    = types.GetBotQRCodeResponse
	GetQRCodeStatusRequest  = types.GetQRCodeStatusRequest
	GetQRCodeStatusResponse = types.GetQRCodeStatusResponse
)

// Constants re-exported from types package.
const (
	MessageTypeNone  = types.MessageTypeNone
	MessageTypeUser  = types.MessageTypeUser
	MessageTypeBot   = types.MessageTypeBot

	MessageItemTypeNone  = types.MessageItemTypeNone
	MessageItemTypeText  = types.MessageItemTypeText
	MessageItemTypeImage = types.MessageItemTypeImage
	MessageItemTypeVoice = types.MessageItemTypeVoice
	MessageItemTypeFile  = types.MessageItemTypeFile
	MessageItemTypeVideo = types.MessageItemTypeVideo

	MessageStateNew         = types.MessageStateNew
	MessageStateGenerating  = types.MessageStateGenerating
	MessageStateFinish      = types.MessageStateFinish

	UploadMediaTypeImage = types.UploadMediaTypeImage
	UploadMediaTypeVideo = types.UploadMediaTypeVideo
	UploadMediaTypeFile  = types.UploadMediaTypeFile
	UploadMediaTypeVoice = types.UploadMediaTypeVoice

	TypingStatusTyping = types.TypingStatusTyping
	TypingStatusCancel = types.TypingStatusCancel

	LoginStatusWaiting  = types.LoginStatusWaiting
	LoginStatusScanned  = types.LoginStatusScanned
	LoginStatusConfirmed = types.LoginStatusConfirmed
	LoginStatusExpired  = types.LoginStatusExpired
	LoginStatusCanceled = types.LoginStatusCanceled
)

// Constructor functions wrap types package functions for backward compatibility.

// NewTextMessage creates a new text message for sending.
func NewTextMessage(toUserID, text, contextToken string) *Message {
	return types.NewTextMessage(toUserID, text, contextToken)
}

// NewImageMessage creates a new image message for sending.
func NewImageMessage(toUserID, contextToken string, imageItem *ImageItem) *Message {
	return types.NewImageMessage(toUserID, contextToken, imageItem)
}

// NewVideoMessage creates a new video message for sending.
func NewVideoMessage(toUserID, contextToken string, videoItem *VideoItem) *Message {
	return types.NewVideoMessage(toUserID, contextToken, videoItem)
}

// NewFileMessage creates a new file message for sending.
func NewFileMessage(toUserID, contextToken string, fileItem *FileItem) *Message {
	return types.NewFileMessage(toUserID, contextToken, fileItem)
}