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

// SimpleBot demonstrates the simplest way to create a WeChat bot.
// Just call Run() - the SDK handles login automatically!
func main() {
	// Setup logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	fmt.Println("=== WeChat iLink SDK - Simple Bot ===")

	// Create token store for persistence (auto-login support)
	tokenStore, _ := login.NewFileTokenStore("")

	// Define QR code display callback
	qrCallback := func(ctx context.Context, qr *login.QRCode) error {
		login.PrintQRCodeWithTerm(qr)
		return nil
	}

	// Create client with login callback
	// Run() will automatically handle login if needed
	client, _ := ilinksdk.NewClient(
		ilinksdk.WithLogger(logger),
		ilinksdk.WithTokenStore(tokenStore),
		ilinksdk.WithOnLogin(qrCallback),
	)
	defer client.Close()

	fmt.Println("Starting bot... (Scan QR code if prompted)")
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

	// That's it! Just call Run() - SDK handles everything:
	// 1. Check if logged in (using stored token)
	// 2. If not logged in, call OnLogin callback to show QR code
	// 3. After login, start processing messages
	// 4. If session expires, call OnSessionExpired callback
	if err := client.Run(ctx, func(ctx context.Context, msg *ilink.Message) error {
		logger.Info("received message", "from", msg.FromUserID)

		// Auto-reply to text messages
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
