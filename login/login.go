// Package login provides authentication functionality for WeChat Bot SDK.
package login

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/skip2/go-qrcode"
	"github.com/the-yex/wechat-ilink-sdk/ilink"
)

// LoginConfig holds login configuration.
type LoginConfig struct {
	// PollInterval is the interval between status checks (default: 2s)
	PollInterval time.Duration
	// QRCodeExpiry is the QR code expiration time (default: 5 minutes)
	QRCodeExpiry time.Duration
	// MaxRefreshCount is the maximum number of QR code refreshes (default: 3)
	MaxRefreshCount int
}

// DefaultLoginConfig returns the default login configuration.
func DefaultLoginConfig() LoginConfig {
	return LoginConfig{
		PollInterval:    2 * time.Second,
		QRCodeExpiry:    5 * time.Minute,
		MaxRefreshCount: 3,
	}
}

// QRCode represents a QR code for login.
type QRCode struct {
	Content   string    // QR code content (URL)
	ImageURL  string    // QR code image URL
	StartedAt time.Time // When the QR code was created (for TTL tracking)
}

// IsExpired checks if the QR code has exceeded its TTL (5 minutes).
func (q *QRCode) IsExpired() bool {
	return time.Since(q.StartedAt) > 5*time.Minute
}

// TerminalString returns a formatted string for terminal display.
// It includes both the QR code image URL and an ASCII QR code for scanning.
func (q *QRCode) TerminalString() string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString("========================================\n")
	sb.WriteString("     SCAN QR CODE TO LOGIN\n")
	sb.WriteString("========================================\n")
	sb.WriteString("\n")

	// Display QR code image URL (primary method)
	if q.ImageURL != "" {
		sb.WriteString("【推荐】打开手机浏览器，访问以下链接：\n")
		sb.WriteString("\n")
		sb.WriteString("  " + q.ImageURL + "\n")
		sb.WriteString("\n")
		sb.WriteString("然后使用微信扫码登录\n")
		sb.WriteString("\n")
		sb.WriteString("----------------------------------------\n")
		sb.WriteString("\n")
	}

	// Display terminal QR code
	if q.ImageURL != "" {
		sb.WriteString("或直接扫描终端中的二维码：\n")
		sb.WriteString("\n")
		sb.WriteString(q.generateQRCodeASCII())
		sb.WriteString("\n")
	}

	sb.WriteString("请使用微信扫码并确认登录\n")
	sb.WriteString("二维码将在 5 分钟后过期\n")
	sb.WriteString("========================================\n")

	return sb.String()
}

// generateQRCodeASCII generates an ASCII QR code string.
func (q *QRCode) generateQRCodeASCII() string {
	if q.ImageURL == "" {
		return ""
	}
	qr, err := qrcode.New(q.ImageURL, qrcode.Medium)
	if err != nil {
		return fmt.Sprintf("Failed to generate QR code: %v\n", err)
	}
	return qr.ToSmallString(true)
}

// LoginFlow manages the QR code login process.
type LoginFlow struct {
	client *ilink.Client
	config LoginConfig

	qrCode       *QRCode
	refreshCount int
}

// NewLoginFlow creates a new login flow.
func NewLoginFlow(client *ilink.Client, config LoginConfig) *LoginFlow {
	return &LoginFlow{
		client: client,
		config: config,
	}
}

// GetQRCode retrieves a new QR code for login.
func (f *LoginFlow) GetQRCode(ctx context.Context) (*QRCode, error) {
	resp, err := f.client.GetBotQRCode(ctx, &ilink.GetBotQRCodeRequest{})
	if err != nil {
		return nil, fmt.Errorf("get QR code: %w", err)
	}
	if resp.Ret != 0 {
		return nil, fmt.Errorf("get QR code failed: %d", resp.Ret)
	}
	qr := &QRCode{
		Content:   resp.QRCode,
		ImageURL:  resp.ImageURL,
		StartedAt: time.Now(), // Track when QR code was created
	}

	f.qrCode = qr
	return qr, nil
}

// PollStatus polls the QR code scan status.
// It returns the login result when the user confirms login.
func (f *LoginFlow) PollStatus(ctx context.Context) (*ilink.LoginResult, error) {
	if f.qrCode == nil {
		return nil, fmt.Errorf("no QR code, call GetQRCode first")
	}

	ticker := time.NewTicker(f.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			result, err := f.checkStatus(ctx)
			if err != nil {
				return nil, err
			}
			if result != nil {
				return result, nil
			}
		}
	}
}

// checkStatus checks the QR code status once.
// Returns nil if waiting, or the login result if confirmed.
func (f *LoginFlow) checkStatus(ctx context.Context) (*ilink.LoginResult, error) {
	resp, err := f.client.GetQRCodeStatus(ctx, &ilink.GetQRCodeStatusRequest{
		QRCode: f.qrCode.Content,
	})
	if err != nil {
		return nil, fmt.Errorf("check status: %w", err)
	}

	switch resp.Status {
	case ilink.LoginStatusWaiting:
		// Still waiting for scan - check if client-side TTL exceeded
		if f.qrCode.IsExpired() {
			if f.refreshCount >= f.config.MaxRefreshCount {
				return nil, fmt.Errorf("QR code expired after %d refreshes", f.refreshCount)
			}
			// Refresh QR code
			_, err := f.GetQRCode(ctx)
			if err != nil {
				return nil, fmt.Errorf("refresh QR code: %w", err)
			}
			f.refreshCount++
		}
		return nil, nil // Continue polling

	case ilink.LoginStatusScanned:
		// User has scanned, waiting for confirmation
		fmt.Println("\n👀 已扫码，在微信继续操作...")
		return nil, nil // Continue polling

	case ilink.LoginStatusConfirmed:
		// Login confirmed!
		return &ilink.LoginResult{
			Token:     resp.BotToken,
			AccountID: resp.ILinkBotID,
			UserID:    resp.ILinkUserID,
			BaseURL:   resp.BaseURL,
		}, nil

	case ilink.LoginStatusExpired:
		// API says expired - refresh if we haven't hit the limit
		if f.refreshCount >= f.config.MaxRefreshCount {
			return nil, fmt.Errorf("QR code expired after %d refreshes", f.refreshCount)
		}
		// Refresh QR code
		_, err := f.GetQRCode(ctx)
		if err != nil {
			return nil, fmt.Errorf("refresh QR code: %w", err)
		}
		f.refreshCount++
		return nil, nil // Continue polling

	case ilink.LoginStatusCanceled:
		return nil, fmt.Errorf("login canceled by user")

	default:
		return nil, fmt.Errorf("unknown status: %s", resp.Status)
	}
}

// Login performs a complete login flow:
// 1. Get QR code
// 2. Display QR code (via callback)
// 3. Poll for scan status
// 4. Return login result
//
// The displayCallback is called with the QR code URL/image for display.
func Login(ctx context.Context, client *ilink.Client, displayCallback func(qr *QRCode) error, config LoginConfig) (*ilink.LoginResult, error) {
	return LoginWithContext(ctx, client, func(_ context.Context, qr *QRCode) error {
		return displayCallback(qr)
	}, config)
}

// QRCodeCallback is a callback function for displaying QR code with context support.
type QRCodeCallback func(ctx context.Context, qr *QRCode) error

// LoginWithContext performs a complete login flow with context-aware callback:
// 1. Get QR code
// 2. Display QR code (via callback with context)
// 3. Poll for scan status
// 4. Return login result
func LoginWithContext(ctx context.Context, client *ilink.Client, displayCallback QRCodeCallback, config LoginConfig) (*ilink.LoginResult, error) {
	flow := NewLoginFlow(client, config)

	// Get initial QR code
	qr, err := flow.GetQRCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("get QR code: %w", err)
	}

	// Display QR code with context
	if displayCallback != nil {
		if err := displayCallback(ctx, qr); err != nil {
			return nil, fmt.Errorf("display QR code: %w", err)
		}
	}

	// Poll for status
	return flow.PollStatus(ctx)
}

// PrintQRCodeWithTerm displays the QR code with both URL and terminal ASCII QR code.
// Users can either open the URL in browser or scan the terminal QR code.
func PrintQRCodeWithTerm(qr *QRCode) {
	fmt.Print(qr.TerminalString())
}
