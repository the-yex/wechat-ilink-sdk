package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/the-yex/wechat-ilink-sdk"
	"github.com/the-yex/wechat-ilink-sdk/login"
)

// QRCodeLoginExample demonstrates QR code login with explicit token persistence helpers.
// It shows how to:
// 1. Create a client without token
// 2. Display QR code for scanning
// 3. Wait for user to scan and confirm
// 4. Save and reload the token with single-account helpers
func main() {
	// Setup logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	fmt.Println("=== WeChat iLink SDK - QR Code Login Example ===")

	tokenStore, err := login.NewFileTokenStore("")
	if err != nil {
		logger.Error("failed to create token store", "error", err)
		os.Exit(1)
	}

	// Step 1: Create client without token
	client, err := ilinksdk.NewClient(
		ilinksdk.WithLogger(logger),
	)
	if err != nil {
		logger.Error("failed to create client", "error", err)
		os.Exit(1)
	}
	defer client.Close()

	// Step 2: Start QR code login
	fmt.Println("Starting QR code login...")
	fmt.Println("Please wait for QR code to be generated...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	result, err := client.Login(ctx, func(ctx context.Context, qr *login.QRCode) error {
		login.PrintQRCodeWithTerm(qr)
		return nil
	})

	if err != nil {
		fmt.Printf("\nLogin failed: %v\n", err)
		logger.Error("login failed", "error", err)
		os.Exit(1)
	}

	// Step 3: Login successful!
	fmt.Println("\n========================================")
	fmt.Println("LOGIN SUCCESSFUL!")
	fmt.Println("========================================")
	fmt.Printf("Account ID: %s\n", result.AccountID)
	fmt.Printf("User ID:  %s\n", result.UserID)
	fmt.Println("========================================")

	token := &login.TokenInfo{
		Token:   result.Token,
		BaseURL: result.BaseURL,
		UserID:  result.UserID,
	}
	if err := login.SaveDefaultToken(tokenStore, token); err != nil {
		logger.Error("save token failed", "error", err)
		os.Exit(1)
	}

	saved, err := login.LoadDefaultToken(tokenStore)
	if err != nil {
		logger.Error("load saved token failed", "error", err)
		os.Exit(1)
	}
	if err := client.RestoreToken(saved); err != nil {
		logger.Error("restore token failed", "error", err)
		os.Exit(1)
	}

	fmt.Println("Token saved to ./.weixin/default.json and restored back into the client.")
	fmt.Printf("Restored user: %s\n", client.CurrentUser().UserID)

	fmt.Println("\n=== Example Complete ===")
	fmt.Println("See examples/basic-bot for automatic login and message handling.")
}
