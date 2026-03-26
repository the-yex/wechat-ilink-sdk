package ilinksdk

import (
	"context"

	"github.com/the-yex/wechat-ilink-sdk/ilink"
	"github.com/the-yex/wechat-ilink-sdk/types"
)

// TextHandler handles text messages.
type TextHandler func(ctx context.Context, msg *ilink.Message, text string) error

// ImageHandler handles image messages.
type ImageHandler func(ctx context.Context, msg *ilink.Message, item *types.ImageItem) error

// VideoHandler handles video messages.
type VideoHandler func(ctx context.Context, msg *ilink.Message, item *types.VideoItem) error

// VoiceHandler handles voice messages.
type VoiceHandler func(ctx context.Context, msg *ilink.Message, item *types.VoiceItem) error

// FileHandler handles file messages.
type FileHandler func(ctx context.Context, msg *ilink.Message, item *types.FileItem) error

// MessageRouter routes messages to handlers based on message type.
// This provides a cleaner API for handling different message types.
type MessageRouter struct {
	textHandler  TextHandler
	imageHandler ImageHandler
	videoHandler VideoHandler
	voiceHandler VoiceHandler
	fileHandler  FileHandler
}

// NewMessageRouter creates a new message router.
func NewMessageRouter() *MessageRouter {
	return &MessageRouter{}
}

// OnText registers a handler for text messages.
func (r *MessageRouter) OnText(handler TextHandler) *MessageRouter {
	r.textHandler = handler
	return r
}

// OnImage registers a handler for image messages.
func (r *MessageRouter) OnImage(handler ImageHandler) *MessageRouter {
	r.imageHandler = handler
	return r
}

// OnVideo registers a handler for video messages.
func (r *MessageRouter) OnVideo(handler VideoHandler) *MessageRouter {
	r.videoHandler = handler
	return r
}

// OnVoice registers a handler for voice messages.
func (r *MessageRouter) OnVoice(handler VoiceHandler) *MessageRouter {
	r.voiceHandler = handler
	return r
}

// OnFile registers a handler for file messages.
func (r *MessageRouter) OnFile(handler FileHandler) *MessageRouter {
	r.fileHandler = handler
	return r
}

// Handler returns a MessageHandler that can be passed to client.Run().
func (r *MessageRouter) Handler() MessageHandler {
	return func(ctx context.Context, msg *ilink.Message) error {
		// Only handle user messages
		if !msg.IsFromUser() {
			return nil
		}

		// Process each item in the message
		for _, item := range msg.ItemList {
			switch item.Type {
			case types.MessageItemTypeText:
				if r.textHandler != nil && item.TextItem != nil {
					if err := r.textHandler(ctx, msg, item.TextItem.Text); err != nil {
						return err
					}
				}

			case types.MessageItemTypeImage:
				if r.imageHandler != nil && item.ImageItem != nil {
					if err := r.imageHandler(ctx, msg, item.ImageItem); err != nil {
						return err
					}
				}

			case types.MessageItemTypeVideo:
				if r.videoHandler != nil && item.VideoItem != nil {
					if err := r.videoHandler(ctx, msg, item.VideoItem); err != nil {
						return err
					}
				}

			case types.MessageItemTypeVoice:
				if r.voiceHandler != nil && item.VoiceItem != nil {
					if err := r.voiceHandler(ctx, msg, item.VoiceItem); err != nil {
						return err
					}
				}

			case types.MessageItemTypeFile:
				if r.fileHandler != nil && item.FileItem != nil {
					if err := r.fileHandler(ctx, msg, item.FileItem); err != nil {
						return err
					}
				}
			}
		}

		return nil
	}
}