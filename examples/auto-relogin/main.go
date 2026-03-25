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
// The Run() method handles everything: login, message processing, and re-login on expiry.
func main() {
	// Setup logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	fmt.Println("=== WeChat iLink SDK - Auto Re-login Demo ===")

	// Create token store for persistence
	tokenStore, _ := login.NewFileTokenStore("")

	// Create client with all callbacks configured
	client, _ := ilinksdk.NewClient(
		ilinksdk.WithLogger(logger),
		ilinksdk.WithTokenStore(tokenStore),
		// Called when QR code login is needed
		ilinksdk.WithOnLogin(func(ctx context.Context, qr *login.QRCode) error {
			login.PrintQRCodeWithTerm(qr)
			return nil
		}),
		// Called when session expires - trigger re-login
		ilinksdk.WithOnSessionExpired(func(ctx context.Context) (*ilink.LoginResult, error) {
			fmt.Println("\n========================================")
			fmt.Println("  Session expired! Please re-scan QR code")
			fmt.Println("========================================")
			// Return nil to trigger QR code login via OnLogin callback
			// Or return client.Login() with a specific callback
			return nil, nil // This will stop the loop; user can restart
		}),
	)
	defer client.Close()

	fmt.Println("Starting bot...")
	fmt.Println("When session expires, you will be prompted to re-scan QR code.")
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

	// Run handles everything:
	// 1. Auto-login if not logged in (using OnLogin callback)
	// 2. Process messages
	// 3. Re-login on session expiry (using OnSessionExpired callback)
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
