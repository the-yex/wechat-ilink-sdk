package service

import (
	"time"

	"github.com/the-yex/wechat-ilink-sdk/ilink"
)

// sessionService implements SessionService.
type sessionService struct {
	apiClient *ilink.Client
}

// NewSessionService creates a new SessionService.
func NewSessionService(api *ilink.Client) SessionService {
	return &sessionService{
		apiClient: api,
	}
}

// IsPaused returns true if the session is paused.
func (s *sessionService) IsPaused() bool {
	return s.apiClient.IsPaused()
}

// RemainingPause returns the remaining pause duration.
func (s *sessionService) RemainingPause() time.Duration {
	return s.apiClient.RemainingPause()
}