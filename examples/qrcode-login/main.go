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

// QRCodeLoginExample demonstrates simple QR code login flow.
// It shows how to:
// 1. Create a client without token
// 2. Display QR code for scanning
// 3. Wait for user to scan and confirm
// 4. Get login result with account info
func main() {
	// Setup logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	fmt.Println("=== WeChat iLink SDK - QR Code Login Example ===")

	// Step 1: Create client without token
	// The client will be used for login, then can be used for sending messages
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

	fmt.Println("\n=== Example Complete ===")
	fmt.Println("You can now use the client to send messages!")
	fmt.Println("See examples/basic-bot for message handling examples.")
}
