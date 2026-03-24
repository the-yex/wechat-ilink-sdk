package service

import (
	"context"

	"github.com/the-yex/wechat-ilink-sdk/media"
)

// mediaService implements MediaService.
type mediaService struct {
	cdnClient *media.Client
}

// NewMediaService creates a new MediaService.
func NewMediaService(cdn *media.Client) MediaService {
	return &mediaService{
		cdnClient: cdn,
	}
}

// Upload uploads a media file to CDN.
func (s *mediaService) Upload(ctx context.Context, req *media.UploadRequest) (*media.UploadResult, error) {
	return s.cdnClient.Upload(ctx, req)
}

// Download downloads and decrypts a media file from CDN.
func (s *mediaService) Download(ctx context.Context, req *media.DownloadRequest) ([]byte, error) {
	return s.cdnClient.Download(ctx, req)
}