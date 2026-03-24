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

// SimpleQRCodeLogin demonstrates the simplest QR code login flow.
// Run this example to see how to authenticate with WeChat using a QR code.
func main() {
	// Create client without token - we will login
	client, err := ilinksdk.NewClient(
		ilinksdk.WithLogger(slog.New(slog.NewTextHandler(os.Stdout, nil))),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create client: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	fmt.Println("=== Simple QR Code Login ===")
	fmt.Println("Generating QR code...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	result, err := client.Login(ctx, func(ctx context.Context, qr *login.QRCode) error {
		login.PrintQRCodeWithTerm(qr)
		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Login failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Login successful!")
	fmt.Printf("Account: %s\n", result.AccountID)
	fmt.Printf("User: %s\n", result.UserID)
}
