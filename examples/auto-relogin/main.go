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
	tokenStore, _ := login.NewFileTokenStore("")

	// Create client - SDK handles login and re-login automatically by default
	client, _ := ilinksdk.NewClient(
		ilinksdk.WithLogger(logger),
		ilinksdk.WithTokenStore(tokenStore),
		// That's it! SDK will:
		// 1. Show QR code in terminal when login needed (default OnLogin)
		// 2. Auto re-login when session expires (default OnSessionExpired)
	)
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

	// Run handles everything automatically
	if err := client.Run(ctx, func(ctx context.Context, msg *ilink.Message) error {
		logger.Info("received message", "from", msg.FromUserID)
		if text := msg.GetText(); text != "" {
			logger.Info("message content", "text", text)
			reply := fmt.Sprintf("[Auto-reply] Received: %s", text)
			if err := client.SendText(ctx, msg.FromUserID, reply); err != nil {
				logger.Error("failed to send reply", "error", err)
			}
		}
		return nil
	}); err != nil && err != context.Canceled {
		logger.Error("bot error", "error", err)
	}

	fmt.Println("Bot stopped.")
}