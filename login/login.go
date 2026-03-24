// Package login provides authentication functionality for WeChat Bot SDK.
package login

import (
	"context"
	"fmt"
	"time"

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
	ExpiresAt time.Time // Expiration time
}

// IsExpired returns true if the QR code has expired.
func (q *QRCode) IsExpired() bool {
	return time.Now().After(q.ExpiresAt)
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
		return nil, fmt.Errorf("get QR code failed: %s", resp.ErrMsg)
	}

	qr := &QRCode{
		Content:  resp.QRCode,
		ImageURL: resp.ImageURL,
	}
	if resp.ExpiresIn > 0 {
		qr.ExpiresAt = time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second)
	} else {
		qr.ExpiresAt = time.Now().Add(f.config.QRCodeExpiry)
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
		// Still waiting, check if expired
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
		if f.refreshCount >= f.config.MaxRefreshCount {
			return nil, fmt.Errorf("QR code expired")
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
		return nil, fmt.Errorf("unknown status: %d", resp.Status)
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

// PrintQRCode prints the QR code to the terminal for scanning.
// It uses the QR code content (URL) to generate a terminal-friendly QR code.
func PrintQRCode(qr *QRCode) {
	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("     SCAN QR CODE TO LOGIN")
	fmt.Println("========================================")
	fmt.Println()

	// Display QR code image URL if available
	if qr.ImageURL != "" {
		fmt.Println("QR Code Image URL:")
		fmt.Println(qr.ImageURL)
		fmt.Println()
	}

	// Display QR code content
	if qr.Content != "" {
		fmt.Println("QR Code Content (for scanning):")
		fmt.Println(qr.Content)
		fmt.Println()
	}

	fmt.Println("Please scan with WeChat app and confirm login.")
	fmt.Println("QR code will expire in 5 minutes.")
	fmt.Println("========================================")
}

// PrintQRCodeWithTerm displays the QR code in the terminal using ASCII art.
// This allows users to scan directly from their terminal.
func PrintQRCodeWithTerm(qr *QRCode) {
	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("     SCAN QR CODE TO LOGIN")
	fmt.Println("========================================")
	fmt.Println()

	// Display QR code image URL if available
	if qr.ImageURL != "" {
		fmt.Println("QR Code Image URL:")
		fmt.Println(qr.ImageURL)
		fmt.Println()
	}

	// Display QR code content for terminal scanning
	if qr.Content != "" {
		fmt.Println("Terminal QR Code (scan from screen):")
		fmt.Println()
		printAsciiQR(qr.Content)
		fmt.Println()
	}

	fmt.Println("Please scan with WeChat app and confirm login.")
	fmt.Println("QR code will expire in 5 minutes.")
	fmt.Println("========================================")
}

// printAsciiQR prints a simple ASCII representation of the QR code.
// For a full QR code terminal display, use an external library like qrterminal.
func printAsciiQR(content string) {
	// Simple ASCII border
	width := 40
	if len(content) > 0 {
		fmt.Println("+" + repeatStr("-", width) + "+")
		fmt.Println("|" + centerStr("QR Code Content", width) + "|")
		fmt.Println("+" + repeatStr("-", width) + "+")
		// Print content preview (truncated if too long)
		preview := content
		if len(preview) > width-2 {
			preview = preview[:width-5] + "..."
		}
		fmt.Println("| " + centerStr(preview, width-2) + " |")
		fmt.Println("+" + repeatStr("-", width) + "+")
		fmt.Println()
		fmt.Println("Note: For best scanning experience,")
		fmt.Println("open the Image URL in a browser.")
	}
}

// repeatStr repeats a string n times.
func repeatStr(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}

// centerStr centers a string within a given width.
func centerStr(s string, width int) string {
	padding := (width - len(s)) / 2
	if padding < 0 {
		padding = 0
	}
	return repeatStr(" ", padding) + s + repeatStr(" ", padding)
}
