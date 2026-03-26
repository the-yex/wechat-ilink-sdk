package media

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/the-yex/wechat-ilink-sdk/ilink"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestClient_Upload(t *testing.T) {
	data := []byte("test image data")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ilink/bot/getuploadurl" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(ilink.GetUploadURLResponse{
				UploadParam: "upload_param_123",
			})
		} else if r.URL.Path == "/upload" {
			// Verify content type
			assert.Equal(t, "application/octet-stream", r.Header.Get("Content-Type"))

			// Return download param in header
			w.Header().Set("x-encrypted-param", "download_param_123")
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	apiClient := ilink.NewClient(ilink.ClientConfig{
		BaseURL: server.URL,
		Token:   "test-token",
	})

	client := NewClient(server.URL+"/", apiClient)

	result, err := client.Upload(context.Background(), &UploadRequest{
		Data:      data,
		MediaType: ilink.UploadMediaTypeImage,
		ToUserID:  "user1",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, result.FileKey)
	assert.NotEmpty(t, result.AESKey)
	assert.Equal(t, "download_param_123", result.DownloadEncryptedQueryParam)
	assert.Equal(t, len(data), result.FileSize)
}

func TestClient_Upload_Error(t *testing.T) {
	t.Run("get upload url error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		apiClient := ilink.NewClient(ilink.ClientConfig{BaseURL: server.URL})
		client := NewClient(server.URL, apiClient)

		_, err := client.Upload(context.Background(), &UploadRequest{
			Data:      []byte("test"),
			MediaType: ilink.UploadMediaTypeImage,
		})

		require.Error(t, err)
	})

	t.Run("cdn upload error", func(t *testing.T) {
		getUploadURLCalled := false
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !getUploadURLCalled {
				getUploadURLCalled = true
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(ilink.GetUploadURLResponse{
					UploadParam: "upload_param",
				})
			} else {
				w.WriteHeader(http.StatusBadRequest)
				w.Header().Set("x-error-message", "invalid request")
			}
		}))
		defer server.Close()

		apiClient := ilink.NewClient(ilink.ClientConfig{BaseURL: server.URL})
		client := NewClient(server.URL, apiClient)

		_, err := client.Upload(context.Background(), &UploadRequest{
			Data:      []byte("test"),
			MediaType: ilink.UploadMediaTypeImage,
		})

		require.Error(t, err)
	})
}

func TestClient_Download(t *testing.T) {
	// Create test data
	plaintext := []byte("test content for download")
	key, _ := GenerateAESKey()
	ciphertext, _ := EncryptAESECB(plaintext, key)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "download")

		w.WriteHeader(http.StatusOK)
		w.Write(ciphertext)
	}))
	defer server.Close()

	apiClient := ilink.NewClient(ilink.ClientConfig{BaseURL: server.URL})
	client := NewClient(server.URL, apiClient)

	// AESKey should be base64 encoded (either raw 16 bytes or hex 32 chars)
	result, err := client.Download(context.Background(), &DownloadRequest{
		EncryptQueryParam: "test_param",
		AESKey:            base64.StdEncoding.EncodeToString(key),
	})

	require.NoError(t, err)
	assert.Equal(t, plaintext, result)
}

func TestClient_Download_Error(t *testing.T) {
	t.Run("http error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		apiClient := ilink.NewClient(ilink.ClientConfig{BaseURL: server.URL})
		client := NewClient(server.URL, apiClient)

		_, err := client.Download(context.Background(), &DownloadRequest{
			EncryptQueryParam: "test_param",
			AESKey:            base64.StdEncoding.EncodeToString([]byte("0123456789abcdef")),
		})

		require.Error(t, err)
	})

	t.Run("invalid aes key", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("encrypted data"))
		}))
		defer server.Close()

		apiClient := ilink.NewClient(ilink.ClientConfig{BaseURL: server.URL})
		client := NewClient(server.URL, apiClient)

		_, err := client.Download(context.Background(), &DownloadRequest{
			EncryptQueryParam: "test_param",
			AESKey:            "invalid-key",
		})

		require.Error(t, err)
	})
}

func TestCDNError(t *testing.T) {
	err := &CDNError{StatusCode: 400, Message: "bad request"}

	assert.Equal(t, "media error: status=400, message=bad request", err.Error())
	assert.True(t, err.IsClientError())
	assert.False(t, err.IsServerError())

	err2 := &CDNError{StatusCode: 500, Message: "server error"}
	assert.False(t, err2.IsClientError())
	assert.True(t, err2.IsServerError())
}

func TestNewClientWithHTTPClient_UsesInjectedHTTPClient(t *testing.T) {
	calls := 0
	injected := &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			assert.Equal(t, "/download", req.URL.Path)
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("payload")),
			}, nil
		}),
	}

	client := NewClientWithHTTPClient("https://cdn.example.com", nil, injected)

	data, err := client.DownloadPlain(context.Background(), "download_param")
	require.NoError(t, err)
	assert.Equal(t, []byte("payload"), data)
	assert.Equal(t, 1, calls)
	assert.NotSame(t, injected, client.httpClient)
}

func keyToHex(key []byte) string {
	hex := make([]byte, len(key)*2)
	for i, b := range key {
		hex[i*2] = "0123456789abcdef"[b>>4]
		hex[i*2+1] = "0123456789abcdef"[b&0x0f]
	}
	return string(hex)
}
