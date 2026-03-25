package login

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/the-yex/wechat-ilink-sdk/ilink"
)

func TestLoginFlow_GetQRCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/ilink/bot/get_bot_qrcode", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ilink.GetBotQRCodeResponse{
			QRCode:   "qrcode123",
			ImageURL: "https://example.com/qr.png",
		})
	}))
	defer server.Close()

	client := ilink.NewClient(ilink.ClientConfig{BaseURL: server.URL})
	flow := NewLoginFlow(client, DefaultLoginConfig())

	qr, err := flow.GetQRCode(context.Background())

	require.NoError(t, err)
	assert.Equal(t, "qrcode123", qr.Content)
	assert.Equal(t, "https://example.com/qr.png", qr.ImageURL)
	assert.False(t, qr.IsExpired())
}

func TestLoginFlow_PollStatus_Confirmed(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")

		if callCount <= 2 {
			// First two calls: waiting
			json.NewEncoder(w).Encode(ilink.GetQRCodeStatusResponse{
				Status: ilink.LoginStatusWaiting,
			})
		} else if callCount == 3 {
			// Third call: scanned
			json.NewEncoder(w).Encode(ilink.GetQRCodeStatusResponse{
				Status: ilink.LoginStatusScanned,
			})
		} else {
			// Fourth call: confirmed
			json.NewEncoder(w).Encode(ilink.GetQRCodeStatusResponse{
				Status:      ilink.LoginStatusConfirmed,
				BotToken:    "bot_token_123",
				ILinkBotID:  "bot123",
				ILinkUserID: "user123",
				BaseURL:     "https://api.example.com",
			})
		}
	}))
	defer server.Close()

	client := ilink.NewClient(ilink.ClientConfig{BaseURL: server.URL})

	// Use shorter poll interval for testing
	config := LoginConfig{
		PollInterval:    10 * time.Millisecond,
		QRCodeExpiry:    5 * time.Minute,
		MaxRefreshCount: 3,
	}

	flow := NewLoginFlow(client, config)

	// Get QR code first
	qr, err := flow.GetQRCode(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, qr)

	// Poll for status
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	result, err := flow.PollStatus(ctx)

	require.NoError(t, err)
	assert.Equal(t, "bot_token_123", result.Token)
	assert.Equal(t, "bot123", result.AccountID)
	assert.Equal(t, "user123", result.UserID)
}

func TestLoginFlow_PollStatus_Canceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ilink.GetQRCodeStatusResponse{
			Status: ilink.LoginStatusWaiting,
		})
	}))
	defer server.Close()

	client := ilink.NewClient(ilink.ClientConfig{BaseURL: server.URL})
	flow := NewLoginFlow(client, LoginConfig{PollInterval: 10 * time.Millisecond})

	qr, err := flow.GetQRCode(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, qr)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = flow.PollStatus(ctx)
	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestLoginFlow_QRCodeExpired(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ilink/bot/get_bot_qrcode" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(ilink.GetBotQRCodeResponse{
				QRCode:   "qrcode123",
				ImageURL: "https://example.com/qr.png",
			})
		} else {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(ilink.GetQRCodeStatusResponse{
				Status: ilink.LoginStatusExpired,
			})
		}
	}))
	defer server.Close()

	client := ilink.NewClient(ilink.ClientConfig{BaseURL: server.URL})

	// Create QR code with already expired StartedAt time
	flow := NewLoginFlow(client, LoginConfig{
		PollInterval:    10 * time.Millisecond,
		MaxRefreshCount: 1,
	})

	// Manually create expired QR code
	flow.qrCode = &QRCode{
		Content:   "qrcode123",
		ImageURL:  "https://example.com/qr.png",
		StartedAt: time.Now().Add(-6 * time.Minute), // 6 minutes ago (expired)
	}

	assert.True(t, flow.qrCode.IsExpired())
}

func TestQRCode_IsExpired(t *testing.T) {
	t.Run("not expired", func(t *testing.T) {
		qr := &QRCode{
			Content:   "test",
			StartedAt: time.Now(),
		}
		assert.False(t, qr.IsExpired())
	})

	t.Run("expired", func(t *testing.T) {
		qr := &QRCode{
			Content:   "test",
			StartedAt: time.Now().Add(-6 * time.Minute), // 6 minutes ago
		}
		assert.True(t, qr.IsExpired())
	})
}

func TestDefaultLoginConfig(t *testing.T) {
	config := DefaultLoginConfig()

	assert.Equal(t, 2*time.Second, config.PollInterval)
	assert.Equal(t, 5*time.Minute, config.QRCodeExpiry)
	assert.Equal(t, 3, config.MaxRefreshCount)
}