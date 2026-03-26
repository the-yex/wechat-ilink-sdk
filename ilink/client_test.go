package ilink

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func jsonHTTPResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestClient_GetUpdates(t *testing.T) {
	tests := []struct {
		name       string
		response   interface{}
		statusCode int
		wantErr    bool
		wantMsgs   int
	}{
		{
			name: "success with messages",
			response: GetUpdatesResponse{
				Ret:           0,
				Messages:      []*Message{{FromUserID: "user1"}, {FromUserID: "user2"}},
				GetUpdatesBuf: "buf123",
			},
			statusCode: http.StatusOK,
			wantErr:    false,
			wantMsgs:   2,
		},
		{
			name: "success empty",
			response: GetUpdatesResponse{
				Ret:           0,
				Messages:      []*Message{},
				GetUpdatesBuf: "buf456",
			},
			statusCode: http.StatusOK,
			wantErr:    false,
			wantMsgs:   0,
		},
		{
			name: "session expired",
			response: GetUpdatesResponse{
				Ret:     0,
				ErrCode: SessionExpiredErrCode,
				ErrMsg:  "session expired",
			},
			statusCode: http.StatusOK,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/ilink/bot/getupdates", r.URL.Path)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			client := NewClient(ClientConfig{
				BaseURL: server.URL,
				Token:   "test-token",
			})

			resp, err := client.GetUpdates(context.Background(), &GetUpdatesRequest{})

			if tt.wantErr {
				require.Error(t, err)
				if tt.response.(GetUpdatesResponse).ErrCode == SessionExpiredErrCode {
					assert.True(t, client.IsPaused())
				}
			} else {
				require.NoError(t, err)
				assert.Len(t, resp.Messages, tt.wantMsgs)
			}
		})
	}
}

func TestClient_SendMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/ilink/bot/sendmessage", r.URL.Path)

		// Decode request
		var req SendMessageRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Verify message
		if len(req.Message.ItemList) > 0 {
			assert.Equal(t, MessageItemTypeText, req.Message.ItemList[0].Type)
			assert.Equal(t, "Hello", req.Message.ItemList[0].TextItem.Text)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"ret": 0})
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		BaseURL: server.URL,
		Token:   "test-token",
	})

	err := client.SendMessage(context.Background(), &SendMessageRequest{
		Message: NewTextMessage("user1", "Hello", "token123"),
	})
	require.NoError(t, err)
}

func TestClient_GetUploadURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/ilink/bot/getuploadurl", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(GetUploadURLResponse{
			UploadParam: "upload_param_123",
		})
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		BaseURL: server.URL,
		Token:   "test-token",
	})

	resp, err := client.GetUploadURL(context.Background(), &GetUploadURLRequest{
		FileKey:    "file123",
		MediaType:  UploadMediaTypeImage,
		RawSize:    1000,
		RawFileMD5: "md5hash",
		FileSize:   1016,
		AESKey:     "0123456789abcdef",
	})

	require.NoError(t, err)
	assert.Equal(t, "upload_param_123", resp.UploadParam)
}

func TestClient_GetConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/ilink/bot/getconfig", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(GetConfigResponse{
			TypingTicket: "typing_ticket_123",
		})
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		BaseURL: server.URL,
		Token:   "test-token",
	})

	resp, err := client.GetConfig(context.Background(), &GetConfigRequest{
		ILinkUserID: "user1",
	})

	require.NoError(t, err)
	assert.Equal(t, "typing_ticket_123", resp.TypingTicket)
}

func TestClient_SendTyping(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/ilink/bot/sendtyping", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"ret": 0})
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		BaseURL: server.URL,
		Token:   "test-token",
	})

	err := client.SendTyping(context.Background(), &SendTypingRequest{
		ILinkUserID:  "user1",
		TypingTicket: "ticket123",
		Status:       int(TypingStatusTyping),
	})

	require.NoError(t, err)
}

func TestClient_GetBotQRCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/ilink/bot/get_bot_qrcode", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(GetBotQRCodeResponse{
			QRCode:   "qrcode_url_123",
			ImageURL: "https://example.com/qr.png",
		})
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		BaseURL: server.URL,
	})

	resp, err := client.GetBotQRCode(context.Background(), &GetBotQRCodeRequest{})

	require.NoError(t, err)
	assert.Equal(t, "qrcode_url_123", resp.QRCode)
	assert.Equal(t, "https://example.com/qr.png", resp.ImageURL)
}

func TestClient_GetQRCodeStatus(t *testing.T) {
	tests := []struct {
		name       string
		status     LoginStatus
		wantStatus LoginStatus
	}{
		{"waiting", LoginStatusWaiting, LoginStatusWaiting},
		{"scanned", LoginStatusScanned, LoginStatusScanned},
		{"confirmed", LoginStatusConfirmed, LoginStatusConfirmed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/ilink/bot/get_qrcode_status", r.URL.Path)

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(GetQRCodeStatusResponse{
					Status:     tt.status,
					BotToken:   "token123",
					ILinkBotID: "bot123",
				})
			}))
			defer server.Close()

			client := NewClient(ClientConfig{
				BaseURL: server.URL,
			})

			resp, err := client.GetQRCodeStatus(context.Background(), &GetQRCodeStatusRequest{
				QRCode: "qrcode123",
			})

			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.Status)
		})
	}
}

func TestNewClient_UsesInjectedHTTPClients(t *testing.T) {
	apiCalls := 0
	longPollCalls := 0

	client := NewClient(ClientConfig{
		BaseURL: "https://api.example.com",
		HTTPClient: &http.Client{
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				apiCalls++
				assert.Equal(t, "/ilink/bot/sendtyping", req.URL.Path)
				return jsonHTTPResponse(http.StatusOK, `{"ret":0}`), nil
			}),
		},
		LongPollHTTPClient: &http.Client{
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				longPollCalls++
				assert.Equal(t, "/ilink/bot/getupdates", req.URL.Path)
				return jsonHTTPResponse(http.StatusOK, `{"ret":0,"messages":[]}`), nil
			}),
		},
	})

	require.NoError(t, client.SendTyping(context.Background(), &SendTypingRequest{}))

	resp, err := client.GetUpdates(context.Background(), &GetUpdatesRequest{})
	require.NoError(t, err)
	assert.Empty(t, resp.Messages)

	assert.Equal(t, 1, apiCalls)
	assert.Equal(t, 1, longPollCalls)
}

func TestNewClient_DerivesLongPollClientFromHTTPClient(t *testing.T) {
	transport := &http.Transport{}
	baseClient := &http.Client{
		Timeout:   1500 * time.Millisecond,
		Transport: transport,
	}

	client := NewClient(ClientConfig{
		BaseURL:         "https://api.example.com",
		HTTPClient:      baseClient,
		LongPollTimeout: 42 * time.Second,
	})

	assert.NotSame(t, baseClient, client.http)
	assert.NotSame(t, client.http, client.httpLP)
	assert.Equal(t, 1500*time.Millisecond, client.http.Timeout)
	assert.Equal(t, 42*time.Second, client.httpLP.Timeout)
	assert.Same(t, transport, client.http.Transport)
	assert.Same(t, transport, client.httpLP.Transport)
}

func TestClient_SessionGuard(t *testing.T) {
	t.Run("pause and check", func(t *testing.T) {
		client := NewClient(ClientConfig{
			BaseURL: "https://example.com",
		})

		assert.False(t, client.IsPaused())

		// Trigger session expiry
		_, _ = client.GetUpdates(context.Background(), &GetUpdatesRequest{})
		// After session expiry error, should be paused
	})

	t.Run("remaining pause", func(t *testing.T) {
		client := NewClient(ClientConfig{
			BaseURL: "https://example.com",
		})

		remaining := client.RemainingPause()
		assert.Equal(t, time.Duration(0), remaining)
	})
}

func TestClient_BuildHeaders(t *testing.T) {
	client := NewClient(ClientConfig{
		BaseURL: "https://example.com",
		Token:   "test-token",
	})

	headers := client.buildHeaders(100)

	assert.Equal(t, "application/json", headers.Get("Content-Type"))
	assert.Equal(t, "Bearer test-token", headers.Get("Authorization"))
	assert.Equal(t, "wechat-bot-sdk-go/1.0", headers.Get("User-Agent"))
	assert.NotEmpty(t, headers.Get("X-WECHAT-UIN"))
}
