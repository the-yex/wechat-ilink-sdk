package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/the-yex/wechat-ilink-sdk"
	"github.com/the-yex/wechat-ilink-sdk/ilink"
	"github.com/the-yex/wechat-ilink-sdk/login"
)

// This example demonstrates automatic re-login when session expires.
func main() {
	// Setup logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	fmt.Println("=== WeChat iLink SDK - Auto Re-login Demo ===")

	// Create token store
	tokenStore, err := login.NewFileTokenStore("")
	if err != nil {
		logger.Error("failed to create token store", "error", err)
		os.Exit(1)
	}

	// Create client first without callback
	client, err := ilinksdk.NewClient(
		ilinksdk.WithLogger(logger),
		ilinksdk.WithTokenStore(tokenStore),
	)
	if err != nil {
		logger.Error("failed to create client", "error", err)
		os.Exit(1)
	}
	defer client.Close()

	// Set the session expired callback using the client
	client.SetOnSessionExpired(func(ctx context.Context) (*ilink.LoginResult, error) {
		// This callback is invoked when session expires
		fmt.Println("\n========================================")
		fmt.Println("  Session expired! Please re-scan QR code")
		fmt.Println("========================================")

		// Perform QR code login again
		return client.Login(ctx, func(ctx context.Context, qr *login.QRCode) error {
			login.PrintQRCodeWithTerm(qr)
			return nil
		})
	})

	// Initial login
	fmt.Println("\nStarting initial login...")

	ctx := context.Background()
	result, err := client.Login(ctx, func(ctx context.Context, qr *login.QRCode) error {
		login.PrintQRCodeWithTerm(qr)
		return nil
	})

	if err != nil {
		fmt.Printf("\nLogin failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n=== Login Successful ===")
	fmt.Printf("Account ID: %s\n", result.AccountID)
	fmt.Printf("User ID:  %s\n", result.UserID)
	fmt.Println("========================")

	// Start message listener
	fmt.Println("\nStarting message listener...")
	fmt.Println("When session expires, you will be prompted to re-scan QR code.")
	fmt.Println("Press Ctrl+C to stop.")

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	runCtx, runCancel := context.WithCancel(context.Background())
	defer runCancel()

	go func() {
		<-sigChan
		fmt.Println("\nShutting down...")
		runCancel()
	}()

	// Run message loop
	// If session expires, OnSessionExpired callback will be invoked automatically
	if err := client.Run(runCtx, func(ctx context.Context, msg *ilink.Message) error {
		logger.Info("received message", "from", msg.FromUserID)
		logger.Info("message content", "text", msg.GetText())

		text := msg.GetText()
		if text != "" {
			reply := fmt.Sprintf("[Auto-reply] Received: %s", text)
			if err := client.SendText(ctx, msg.FromUserID, reply); err != nil {
				logger.Error("failed to send reply", "error", err)
			}
		}

		return nil
	}); err != nil && err != context.Canceled {
		logger.Error("run error", "error", err)
	}

	fmt.Println("Bot stopped.")
}
