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

// messageHandlers holds handlers for different message types.
type messageHandlers struct {
	messageHandler MessageHandler
	textHandler    TextHandler
	imageHandler   ImageHandler
	videoHandler   VideoHandler
	voiceHandler   VoiceHandler
	fileHandler    FileHandler
}

// hasAnyHandler returns true if any type handler is registered.
func (h *messageHandlers) hasAnyHandler() bool {
	return h.messageHandler != nil || h.textHandler != nil || h.imageHandler != nil || h.videoHandler != nil || h.voiceHandler != nil || h.fileHandler != nil
}

// buildHandler creates a MessageHandler from the registered type handlers.
func (h *messageHandlers) buildHandler() MessageHandler {
	// If a general message handler is set, use it
	if h.messageHandler != nil {
		return h.messageHandler
	}

	// Otherwise, build from type-specific handlers
	return func(ctx context.Context, msg *ilink.Message) error {
		// Only handle user messages
		if !msg.IsFromUser() {
			return nil
		}

		// Process each item in the message
		for _, item := range msg.ItemList {
			switch item.Type {
			case types.MessageItemTypeText:
				if h.textHandler != nil && item.TextItem != nil {
					if err := h.textHandler(ctx, msg, item.TextItem.Text); err != nil {
						return err
					}
				}

			case types.MessageItemTypeImage:
				if h.imageHandler != nil && item.ImageItem != nil {
					if err := h.imageHandler(ctx, msg, item.ImageItem); err != nil {
						return err
					}
				}

			case types.MessageItemTypeVideo:
				if h.videoHandler != nil && item.VideoItem != nil {
					if err := h.videoHandler(ctx, msg, item.VideoItem); err != nil {
						return err
					}
				}

			case types.MessageItemTypeVoice:
				if h.voiceHandler != nil && item.VoiceItem != nil {
					if err := h.voiceHandler(ctx, msg, item.VoiceItem); err != nil {
						return err
					}
				}

			case types.MessageItemTypeFile:
				if h.fileHandler != nil && item.FileItem != nil {
					if err := h.fileHandler(ctx, msg, item.FileItem); err != nil {
						return err
					}
				}
			}
		}

		return nil
	}
}