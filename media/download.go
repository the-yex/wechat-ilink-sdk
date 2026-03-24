package media

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// DownloadRequest represents a media download request.
type DownloadRequest struct {
	EncryptQueryParam string // CDN encrypted query parameter
	AESKey            string // Base64 or hex encoded AES key
}

// Download downloads and decrypts a media file from CDN.
func (c *Client) Download(ctx context.Context, req *DownloadRequest) ([]byte, error) {
	// Parse AES key
	aesKey, err := parseAESKey(req.AESKey)
	if err != nil {
		return nil, fmt.Errorf("parse AES key: %w", err)
	}

	// Build download URL
	downloadURL := fmt.Sprintf("%sdownload?encrypted_query_param=%s",
		c.baseURL,
		url.QueryEscape(req.EncryptQueryParam))

	// Create request
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, &CDNError{StatusCode: resp.StatusCode, Message: string(body)}
	}

	// Read encrypted data
	encrypted, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// Decrypt
	plaintext, err := DecryptAESECB(encrypted, aesKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt data: %w", err)
	}

	return plaintext, nil
}

// DownloadPlain downloads unencrypted data from CDN.
func (c *Client) DownloadPlain(ctx context.Context, encryptQueryParam string) ([]byte, error) {
	downloadURL := fmt.Sprintf("%sdownload?encrypted_query_param=%s",
		c.baseURL,
		url.QueryEscape(encryptQueryParam))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, &CDNError{StatusCode: resp.StatusCode, Message: string(body)}
	}

	return io.ReadAll(resp.Body)
}

// parseAESKey parses an AES key from base64 or hex encoding.
// Supports two formats:
// 1. base64(raw 16 bytes) - direct decode
// 2. base64(hex 32 chars) - decode then hex decode
func parseAESKey(encoded string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("base64 decode: %w", err)
	}

	// If it's 16 bytes directly, use it
	if len(decoded) == 16 {
		return decoded, nil
	}

	// If it's 32 characters and looks like hex, convert
	if len(decoded) == 32 {
		// Check if it's valid hex
		if isHexString(string(decoded)) {
			key, err := hex.DecodeString(string(decoded))
			if err != nil {
				return nil, fmt.Errorf("hex decode: %w", err)
			}
			return key, nil
		}
	}

	return nil, fmt.Errorf("AES key must decode to 16 raw bytes or 32-char hex string")
}

// isHexString checks if a string is a valid hex string.
func isHexString(s string) bool {
	for _, c := range s {
		if !isHexDigit(c) {
			return false
		}
	}
	return true
}

func isHexDigit(c rune) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

// BuildCDNDownloadURL builds a CDN download URL.
func BuildCDNDownloadURL(baseURL, encryptQueryParam string) string {
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	return fmt.Sprintf("%sdownload?encrypted_query_param=%s", baseURL, url.QueryEscape(encryptQueryParam))
}

// BuildCDNUploadURL builds a CDN upload URL.
func BuildCDNUploadURL(baseURL, uploadParam, fileKey string) string {
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	return fmt.Sprintf("%supload?encrypted_query_param=%s&filekey=%s",
		baseURL, url.QueryEscape(uploadParam), url.QueryEscape(fileKey))
}