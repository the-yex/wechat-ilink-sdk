package media

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/the-yex/wechat-ilink-sdk/ilink"
)

const (
	// DefaultCDNBaseURL is the default CDN base URL.
	DefaultCDNBaseURL = "https://novac2c.cdn.weixin.qq.com/c2c"
)

// Client is the CDN client for media upload/download.
type Client struct {
	baseURL    string
	httpClient *http.Client
	apiClient  *ilink.Client
}

// NewClient creates a new CDN client.
func NewClient(baseURL string, apiClient *ilink.Client) *Client {
	if baseURL == "" {
		baseURL = DefaultCDNBaseURL
	}
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        50,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		apiClient: apiClient,
	}
}

// UploadRequest represents a media upload request.
type UploadRequest struct {
	Data      []byte               // File data to upload
	MediaType ilink.UploadMediaType // Media type (IMAGE, VIDEO, FILE, VOICE)
	ToUserID  string               // Target user ID
	AESKey    []byte               // Optional; generated if nil
}

// UploadResult contains upload result information.
type UploadResult struct {
	FileKey                     string // File key (hex encoded 16 bytes)
	DownloadEncryptedQueryParam string // CDN download parameter
	AESKey                      []byte // AES key used for encryption
	FileSize                    int    // Plaintext file size
	FileSizeCiphertext          int    // Ciphertext file size
}

// AESKeyBase64 returns the AES key as base64 encoded hex string.
// This is the format expected by WeChat API for CDNMedia.aes_key.
func (r *UploadResult) AESKeyBase64() string {
	// First convert to hex string, then base64 encode
	// This matches the TS implementation: Buffer.from(aeskey.toString("hex")).toString("base64")
	return base64.StdEncoding.EncodeToString([]byte(hex.EncodeToString(r.AESKey)))
}

// Upload uploads a media file to CDN with AES encryption.
func (c *Client) Upload(ctx context.Context, req *UploadRequest) (*UploadResult, error) {
	if len(req.Data) == 0 {
		return nil, fmt.Errorf("no data to upload")
	}

	// Generate or use provided AES key
	aesKey := req.AESKey
	var err error
	if aesKey == nil {
		aesKey, err = GenerateAESKey()
		if err != nil {
			return nil, fmt.Errorf("generate AES key: %w", err)
		}
	}
	if len(aesKey) != aesBlockSize {
		return nil, fmt.Errorf("AES key must be 16 bytes")
	}

	// Calculate MD5
	hash := md5.Sum(req.Data)
	rawFileMD5 := hex.EncodeToString(hash[:])

	// Calculate sizes
	rawSize := len(req.Data)
	fileSizeCiphertext := AESECBPaddedSize(rawSize)

	// Generate file key
	fileKey, err := GenerateFileKey()
	if err != nil {
		return nil, fmt.Errorf("generate file key: %w", err)
	}

	// Get upload URL from API
	uploadURLResp, err := c.apiClient.GetUploadURL(ctx, &ilink.GetUploadURLRequest{
		FileKey:     fileKey,
		MediaType:   req.MediaType,
		ToUserID:    req.ToUserID,
		RawSize:     rawSize,
		RawFileMD5:  rawFileMD5,
		FileSize:    fileSizeCiphertext,
		NoNeedThumb: true,
		AESKey:      hex.EncodeToString(aesKey),
	})
	if err != nil {
		return nil, fmt.Errorf("get upload URL: %w", err)
	}

	if uploadURLResp.UploadParam == "" {
		return nil, fmt.Errorf("empty upload param")
	}

	// Encrypt the data
	ciphertext, err := EncryptAESECB(req.Data, aesKey)
	if err != nil {
		return nil, fmt.Errorf("encrypt data: %w", err)
	}

	// Upload to CDN
	downloadParam, err := c.uploadToCDN(ctx, uploadURLResp.UploadParam, fileKey, ciphertext)
	if err != nil {
		return nil, fmt.Errorf("upload to CDN: %w", err)
	}

	return &UploadResult{
		FileKey:                     fileKey,
		DownloadEncryptedQueryParam: downloadParam,
		AESKey:                      aesKey,
		FileSize:                    rawSize,
		FileSizeCiphertext:          fileSizeCiphertext,
	}, nil
}

// uploadToCDN uploads encrypted data to CDN.
func (c *Client) uploadToCDN(ctx context.Context, uploadParam, fileKey string, ciphertext []byte) (string, error) {
	// Build CDN upload URL
	uploadURL := fmt.Sprintf("%supload?encrypted_query_param=%s&filekey=%s",
		c.baseURL,
		url.QueryEscape(uploadParam),
		url.QueryEscape(fileKey))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, bytes.NewReader(ciphertext))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	// Handle client errors (4xx) - don't retry
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		errMsg := resp.Header.Get("x-error-message")
		if errMsg == "" {
			body, _ := io.ReadAll(resp.Body)
			errMsg = string(body)
		}
		return "", &CDNError{StatusCode: resp.StatusCode, Message: errMsg}
	}

	// Handle server errors (5xx)
	if resp.StatusCode >= 500 {
		return "", &CDNError{StatusCode: resp.StatusCode, Message: "server error"}
	}

	// Get download param from response header
	downloadParam := resp.Header.Get("x-encrypted-param")
	if downloadParam == "" {
		return "", fmt.Errorf("CDN response missing x-encrypted-param header")
	}

	return downloadParam, nil
}
