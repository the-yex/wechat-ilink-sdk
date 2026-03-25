package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/the-yex/wechat-ilink-sdk/ilink"
	"github.com/the-yex/wechat-ilink-sdk/media"
	"github.com/the-yex/wechat-ilink-sdk/middleware"
)

// Errors returned by MessageService.
var (
	ErrContextTokenRequired = errors.New("context token is required")
)

// messageService implements MessageService.
type messageService struct {
	apiClient     APIClient
	cdnClient     CDNClient
	contextTokens ContextTokenService
	middleware    []middleware.Middleware
}

// NewMessageService creates a new MessageService.
func NewMessageService(
	api APIClient,
	cdn CDNClient,
	tokens ContextTokenService,
	mw []middleware.Middleware,
) MessageService {
	return &messageService{
		apiClient:     api,
		cdnClient:     cdn,
		contextTokens: tokens,
		middleware:    mw,
	}
}

// SendMessage sends a message with the given request.
func (s *messageService) SendMessage(ctx context.Context, req *ilink.SendMessageRequest) error {
	// Apply middleware chain
	handler := middleware.Chain(
		func(ctx context.Context, req *ilink.SendMessageRequest) error {
			return s.apiClient.SendMessage(ctx, req)
		},
		s.middleware...,
	)
	return handler(ctx, req)
}

// SendText sends a text message to a user.
func (s *messageService) SendText(ctx context.Context, toUserID, text string) error {
	contextToken := s.contextTokens.Get("", toUserID)
	if contextToken == "" {
		return ErrContextTokenRequired
	}

	return s.SendMessage(ctx, &ilink.SendMessageRequest{
		Message: ilink.NewTextMessage(toUserID, text, contextToken),
	})
}

// SendImage sends an image message to a user.
func (s *messageService) SendImage(ctx context.Context, toUserID string, imageData []byte) error {
	contextToken := s.contextTokens.Get("", toUserID)
	if contextToken == "" {
		return ErrContextTokenRequired
	}

	// Upload image to CDN
	result, err := s.cdnClient.Upload(ctx, &media.UploadRequest{
		Data:      imageData,
		MediaType: ilink.UploadMediaTypeImage,
		ToUserID:  toUserID,
	})
	if err != nil {
		return fmt.Errorf("upload image: %w", err)
	}

	// Send image message
	return s.SendMessage(ctx, &ilink.SendMessageRequest{
		Message: ilink.NewImageMessage(toUserID, contextToken, &ilink.ImageItem{
			Media: &ilink.CDNMedia{
				EncryptQueryParam: result.DownloadEncryptedQueryParam,
				AESKey:            result.AESKeyBase64(),
				EncryptType:       int(ilink.EncryptTypePackMedia),
			},
			MidSize: result.FileSizeCiphertext,
		}),
	})
}

// SendVideo sends a video message to a user.
func (s *messageService) SendVideo(ctx context.Context, toUserID string, videoData []byte) error {
	contextToken := s.contextTokens.Get("", toUserID)
	if contextToken == "" {
		return ErrContextTokenRequired
	}

	// Upload video to CDN
	result, err := s.cdnClient.Upload(ctx, &media.UploadRequest{
		Data:      videoData,
		MediaType: ilink.UploadMediaTypeVideo,
		ToUserID:  toUserID,
	})
	if err != nil {
		return fmt.Errorf("upload video: %w", err)
	}

	// Send video message
	return s.SendMessage(ctx, &ilink.SendMessageRequest{
		Message: ilink.NewVideoMessage(toUserID, contextToken, &ilink.VideoItem{
			Media: &ilink.CDNMedia{
				EncryptQueryParam: result.DownloadEncryptedQueryParam,
				AESKey:            result.AESKeyBase64(),
				EncryptType:       int(ilink.EncryptTypePackMedia),
			},
			VideoSize: result.FileSizeCiphertext,
		}),
	})
}

// SendVoice sends a voice message to a user.
// voiceItem should contain playtime, encode_type, bits_per_sample, sample_rate from the original message.
func (s *messageService) SendVoice(ctx context.Context, toUserID string, voiceData []byte, voiceItem *ilink.VoiceItem) error {
	contextToken := s.contextTokens.Get("", toUserID)
	if contextToken == "" {
		return ErrContextTokenRequired
	}

	// Upload voice to CDN
	result, err := s.cdnClient.Upload(ctx, &media.UploadRequest{
		Data:      voiceData,
		MediaType: ilink.UploadMediaTypeVoice,
		ToUserID:  toUserID,
	})
	if err != nil {
		return fmt.Errorf("upload voice: %w", err)
	}

	// Build voice message with original encoding parameters
	item := &ilink.VoiceItem{
		Media: &ilink.CDNMedia{
			EncryptQueryParam: result.DownloadEncryptedQueryParam,
			AESKey:            result.AESKeyBase64(),
			EncryptType:       int(ilink.EncryptTypePackMedia),
		},
	}

	// Copy encoding parameters from original voice item
	if voiceItem != nil {
		item.Playtime = voiceItem.Playtime
		item.EncodeType = voiceItem.EncodeType
		item.BitsPerSample = voiceItem.BitsPerSample
		item.SampleRate = voiceItem.SampleRate
	}

	return s.SendMessage(ctx, &ilink.SendMessageRequest{
		Message: ilink.NewVoiceMessage(toUserID, contextToken, item),
	})
}

// SendFile sends a file message to a user.
func (s *messageService) SendFile(ctx context.Context, toUserID, fileName string, fileData []byte) error {
	contextToken := s.contextTokens.Get("", toUserID)
	if contextToken == "" {
		return ErrContextTokenRequired
	}

	// Upload file to CDN
	result, err := s.cdnClient.Upload(ctx, &media.UploadRequest{
		Data:      fileData,
		MediaType: ilink.UploadMediaTypeFile,
		ToUserID:  toUserID,
	})
	if err != nil {
		return fmt.Errorf("upload file: %w", err)
	}

	// Send file message
	return s.SendMessage(ctx, &ilink.SendMessageRequest{
		Message: ilink.NewFileMessage(toUserID, contextToken, &ilink.FileItem{
			FileName: fileName,
			Media: &ilink.CDNMedia{
				EncryptQueryParam: result.DownloadEncryptedQueryParam,
				AESKey:            result.AESKeyBase64(),
				EncryptType:       int(ilink.EncryptTypePackMedia),
			},
			Len: fmt.Sprintf("%d", result.FileSize),
		}),
	})
}

// SendTyping sends a typing indicator to a user.
func (s *messageService) SendTyping(ctx context.Context, toUserID string, typing bool) error {
	// Get config to obtain typing ticket
	config, err := s.apiClient.GetConfig(ctx, &ilink.GetConfigRequest{
		ILinkUserID: toUserID,
	})
	if err != nil {
		return fmt.Errorf("get config: %w", err)
	}

	status := ilink.TypingStatusCancel
	if typing {
		status = ilink.TypingStatusTyping
	}

	return s.apiClient.SendTyping(ctx, &ilink.SendTypingRequest{
		ILinkUserID:  toUserID,
		TypingTicket: config.TypingTicket,
		Status:       int(status),
	})
}
