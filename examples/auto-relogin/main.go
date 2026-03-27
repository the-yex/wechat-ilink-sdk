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
// The SDK handles everything automatically: login, message processing, and re-login on expiry.
func main() {
	// Setup logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	fmt.Println("=== WeChat iLink SDK - Auto Re-login Demo ===")

	// Create token store for persistence
	tokenStore, err := login.NewFileTokenStore("")
	if err != nil {
		logger.Error("create token store", "error", err)
		os.Exit(1)
	}

	var client *ilinksdk.Client
	client, err = ilinksdk.NewClient(
		ilinksdk.WithLogger(logger),
		ilinksdk.WithTokenStore(tokenStore),
		ilinksdk.WithOnSessionExpired(func(ctx context.Context) (*ilink.LoginResult, error) {
			logger.Warn("session expired, asking user to scan again")
			return client.Login(ctx, func(ctx context.Context, qr *login.QRCode) error {
				login.PrintQRCodeWithTerm(qr)
				return nil
			})
		}),
	)
	if err != nil {
		logger.Error("create client", "error", err)
		os.Exit(1)
	}
	defer client.Close()

	fmt.Println("Starting bot...")
	fmt.Println("SDK will automatically handle login and re-login on session expiry.")
	fmt.Println("Press Ctrl+C to stop.")

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		<-sigChan
		fmt.Println("\nShutting down...")
		cancel()
	}()

	client.OnText(func(ctx context.Context, msg *ilink.Message, text string) error {
		logger.Info("received text message", "from", msg.FromUserID, "text", text)
		return client.SendText(ctx, msg.FromUserID, fmt.Sprintf("[Auto-reply] Received: %s", text))
	})

	// Run handles login, token recovery, and our custom session-expired callback.
	if err := client.Run(ctx, nil); err != nil && err != context.Canceled {
		logger.Error("bot error", "error", err)
	}

	fmt.Println("Bot stopped.")
}
