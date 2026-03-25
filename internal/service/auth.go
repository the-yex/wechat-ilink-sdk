package service

import (
	"context"
	"fmt"
	"time"

	"github.com/the-yex/wechat-ilink-sdk/ilink"
	"github.com/the-yex/wechat-ilink-sdk/login"
	"github.com/the-yex/wechat-ilink-sdk/media"
)

// TokenUpdateCallback is called when token is updated.
// This allows the client to update its apiClient and cdnClient.
type TokenUpdateCallback func(token, baseURL, accountID, userID string)

// authService implements AuthService.
type authService struct {
	apiClient  *ilink.Client
	cdnClient  *media.Client
	tokenStore login.TokenStore
	config     *AuthConfig

	// Callback when token is updated
	onTokenUpdate TokenUpdateCallback
}

// AuthConfig holds configuration for AuthService.
// This is a subset of the main Config to avoid circular dependency.
type AuthConfig struct {
	BaseURL         string
	CDNBaseURL      string
	Token           string
	Timeout         time.Duration
	LongPollTimeout time.Duration
}

// NewAuthService creates a new AuthService.
func NewAuthService(
	api *ilink.Client,
	cdn *media.Client,
	store login.TokenStore,
	cfg *AuthConfig,
	onUpdate TokenUpdateCallback,
) AuthService {
	return &authService{
		apiClient:     api,
		cdnClient:     cdn,
		tokenStore:    store,
		config:        cfg,
		onTokenUpdate: onUpdate,
	}
}

// Login performs QR code login.
func (s *authService) Login(ctx context.Context, displayCallback login.QRCodeCallback) (*ilink.LoginResult, error) {
	result, err := login.LoginWithContext(ctx, s.apiClient, displayCallback, login.DefaultLoginConfig())
	if err != nil {
		return nil, err
	}

	// Use default account ID for single-account mode
	accountID := login.DefaultAccountID

	// Update token with full user info
	s.SetToken(result.Token, result.BaseURL, accountID, result.UserID)

	// Save token if store is available
	if s.tokenStore != nil {
		_ = s.tokenStore.Save(accountID, &login.TokenInfo{
			Token:   result.Token,
			BaseURL: result.BaseURL,
			UserID:  result.UserID,
			SavedAt: time.Now().Format(time.RFC3339),
		})
	}

	// Update result with correct account ID
	result.AccountID = accountID
	return result, nil
}

// SetToken sets the authentication token.
func (s *authService) SetToken(token, baseURL, accountID, userID string) {
	s.config.Token = token
	if baseURL != "" {
		s.config.BaseURL = baseURL
	}

	// Notify client to update apiClient and cdnClient
	if s.onTokenUpdate != nil {
		s.onTokenUpdate(token, baseURL, accountID, userID)
	}
}

// LoadToken loads a stored token for an account.
func (s *authService) LoadToken(accountID string) error {
	if s.tokenStore == nil {
		return fmt.Errorf("no token store configured")
	}

	token, err := s.tokenStore.Load(accountID)
	if err != nil {
		return fmt.Errorf("load token: %w", err)
	}
	if token == nil {
		return fmt.Errorf("no token found for account %s", accountID)
	}
	s.SetToken(token.Token, token.BaseURL, accountID, token.UserID)
	return nil
}

// GetCurrentUser returns the current logged-in user info.
func (s *authService) GetCurrentUser() *ilink.LoginResult {
	if s.config.Token == "" {
		return nil
	}
	return &ilink.LoginResult{
		Token:     s.config.Token,
		AccountID: "", // Not available from config alone
		UserID:    "", // Not available from config alone
		BaseURL:   s.config.BaseURL,
	}
}
