package service

import (
	"context"
	"errors"
	"testing"

	"github.com/the-yex/wechat-ilink-sdk/ilink"
	"github.com/the-yex/wechat-ilink-sdk/media"
)

// mockAPIClient mocks APIClient for testing
type mockAPIClient struct {
	sendMessageErr error
	sendTypingErr  error
	getConfigErr   error
	config         *ilink.GetConfigResponse
}

func (m *mockAPIClient) SendMessage(ctx context.Context, req *ilink.SendMessageRequest) error {
	return m.sendMessageErr
}

func (m *mockAPIClient) SendTyping(ctx context.Context, req *ilink.SendTypingRequest) error {
	return m.sendTypingErr
}

func (m *mockAPIClient) GetConfig(ctx context.Context, req *ilink.GetConfigRequest) (*ilink.GetConfigResponse, error) {
	if m.getConfigErr != nil {
		return nil, m.getConfigErr
	}
	if m.config != nil {
		return m.config, nil
	}
	return &ilink.GetConfigResponse{TypingTicket: "test-ticket"}, nil
}

// mockCDNClient mocks CDNClient for testing
type mockCDNClient struct {
	uploadErr     error
	downloadErr   error
	uploadResult  *media.UploadResult
	downloadData  []byte
}

func (m *mockCDNClient) Upload(ctx context.Context, req *media.UploadRequest) (*media.UploadResult, error) {
	if m.uploadErr != nil {
		return nil, m.uploadErr
	}
	if m.uploadResult != nil {
		return m.uploadResult, nil
	}
	return &media.UploadResult{
		DownloadEncryptedQueryParam: "test-param",
		AESKey:                     []byte("1234567890123456"),
	}, nil
}

func (m *mockCDNClient) Download(ctx context.Context, req *media.DownloadRequest) ([]byte, error) {
	if m.downloadErr != nil {
		return nil, m.downloadErr
	}
	return m.downloadData, nil
}

// mockContextTokenService mocks ContextTokenService for testing
type mockContextTokenService struct {
	tokens map[string]string
}

func newMockContextTokenService() *mockContextTokenService {
	return &mockContextTokenService{tokens: make(map[string]string)}
}

func (m *mockContextTokenService) Get(accountID, userID string) string {
	return m.tokens[accountID+":"+userID]
}

func (m *mockContextTokenService) Set(accountID, userID, token string) {
	m.tokens[accountID+":"+userID] = token
}

func (m *mockContextTokenService) Delete(accountID, userID string) {
	delete(m.tokens, accountID+":"+userID)
}

func TestMessageService_SendText(t *testing.T) {
	tests := []struct {
		name         string
		toUserID     string
		text         string
		contextToken string
		sendErr      error
		wantErr      error
	}{
		{
			name:         "success",
			toUserID:     "user123",
			text:         "Hello",
			contextToken: "token123",
			wantErr:      nil,
		},
		{
			name:         "no context token",
			toUserID:     "user123",
			text:         "Hello",
			contextToken: "",
			wantErr:      ErrContextTokenRequired,
		},
		{
			name:         "send message error",
			toUserID:     "user123",
			text:         "Hello",
			contextToken: "token123",
			sendErr:      errors.New("network error"),
			wantErr:      errors.New("network error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTokens := newMockContextTokenService()

			if tt.contextToken != "" {
				mockTokens.Set("", tt.toUserID, tt.contextToken)
			}

			svc := NewMessageService(
				&mockAPIClient{sendMessageErr: tt.sendErr},
				&mockCDNClient{},
				mockTokens,
				nil,
			)

			err := svc.SendText(context.Background(), tt.toUserID, tt.text)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
				} else if !errors.Is(err, tt.wantErr) && err.Error() != tt.wantErr.Error() {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

func TestMessageService_SendImage(t *testing.T) {
	tests := []struct {
		name         string
		toUserID     string
		imageData    []byte
		contextToken string
		uploadErr    error
		wantErr      bool
	}{
		{
			name:         "success",
			toUserID:     "user123",
			imageData:    []byte("fake-image"),
			contextToken: "token123",
			wantErr:      false,
		},
		{
			name:         "no context token",
			toUserID:     "user123",
			imageData:    []byte("fake-image"),
			contextToken: "",
			wantErr:      true,
		},
		{
			name:         "upload error",
			toUserID:     "user123",
			imageData:    []byte("fake-image"),
			contextToken: "token123",
			uploadErr:    errors.New("upload failed"),
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTokens := newMockContextTokenService()

			if tt.contextToken != "" {
				mockTokens.Set("", tt.toUserID, tt.contextToken)
			}

			svc := NewMessageService(
				&mockAPIClient{},
				&mockCDNClient{uploadErr: tt.uploadErr},
				mockTokens,
				nil,
			)

			err := svc.SendImage(context.Background(), tt.toUserID, tt.imageData)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

func TestMessageService_SendTyping(t *testing.T) {
	tests := []struct {
		name        string
		typing      bool
		getConfigErr error
		wantErr     bool
	}{
		{
			name:    "typing true",
			typing:  true,
			wantErr: false,
		},
		{
			name:    "typing false",
			typing:  false,
			wantErr: false,
		},
		{
			name:        "get config error",
			typing:      true,
			getConfigErr: errors.New("config error"),
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewMessageService(
				&mockAPIClient{getConfigErr: tt.getConfigErr},
				&mockCDNClient{},
				newMockContextTokenService(),
				nil,
			)

			err := svc.SendTyping(context.Background(), "user123", tt.typing)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}