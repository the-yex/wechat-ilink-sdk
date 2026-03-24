package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/the-yex/wechat-ilink-sdk"
	"github.com/the-yex/wechat-ilink-sdk/ilink"
	"github.com/the-yex/wechat-ilink-sdk/login"
)

// QRCodeLoginWithAutoReply demonstrates QR code login with a simple auto-reply bot.
func main() {
	// Setup logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	fmt.Println("=== WeChat iLink SDK - QR Code Login with Auto-Reply ===")

	// Step 1: Create file-based token store for persistence
	tokenStore, err := login.NewFileTokenStore("")
	if err != nil {
		logger.Error("failed to create token store", "error", err)
		os.Exit(1)
	}

	// Step 2: Create client with token store
	client, err := ilinksdk.NewClient(
		ilinksdk.WithLogger(logger),
		ilinksdk.WithTokenStore(tokenStore),
	)
	if err != nil {
		logger.Error("failed to create client", "error", err)
		os.Exit(1)
	}
	defer client.Close()

	// Step 3: Check for stored token
	accounts, err := client.ListAccounts()
	if err == nil && len(accounts) > 0 {
		fmt.Printf("Found stored account: %s\n", accounts[0])
		fmt.Println("Attempting to login with stored token...")

		if err := client.LoadToken(accounts[0]); err != nil {
			fmt.Printf("Failed to load stored token: %v\n", err)
			fmt.Println("Proceeding with QR code login...")
		} else {
			fmt.Println("Successfully loaded stored token!")
		}
	}

	// Step 4: Start QR code login (if not already logged in)
	fmt.Println("\nStarting QR code login...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	result, err := client.Login(ctx, func(ctx context.Context, qr *login.QRCode) error {
		login.PrintQRCodeWithTerm(qr)
		return nil
	})

	if err != nil {
		fmt.Printf("\nLogin failed: %v\n", err)
		os.Exit(1)
	}

	// Step 5: Login successful
	fmt.Println("\n=== Login Successful ===")
	fmt.Printf("Account ID: %s\n", result.AccountID)
	fmt.Printf("User ID:  %s\n", result.UserID)
	fmt.Println("========================")

	// Step 6: Start simple message listener
	fmt.Println("Starting message listener...")
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
	if err := client.Run(runCtx, func(ctx context.Context, msg *ilink.Message) error {
		logger.Info("received message",
			"from", msg.FromUserID,
		)

		// Auto-reply to text messages
		text := msg.GetText()
		if text != "" {
			logger.Info("message content", "text", text)

			reply := fmt.Sprintf("[Auto-reply] I received your message: %s", text)
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
