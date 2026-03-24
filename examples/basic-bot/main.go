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
	"github.com/the-yex/wechat-ilink-sdk/middleware"
)

func main() {
	// Setup logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Create file-based token store
	tokenStore, err := login.NewFileTokenStore("")
	if err != nil {
		logger.Error("create token store", "error", err)
		os.Exit(1)
	}

	// Create client (without token - will login)
	client, err := ilinksdk.NewClient(
		ilinksdk.WithLogger(logger),
		ilinksdk.WithTokenStore(tokenStore),
		ilinksdk.WithMiddleware(
			middleware.Logging(logger),
			middleware.Recovery(logger),
			middleware.Retry(middleware.DefaultRetryConfig()),
		),
	)
	if err != nil {
		logger.Error("create client", "error", err)
		os.Exit(1)
	}
	defer client.Close()

	// Check if we have stored tokens
	accounts, err := client.ListAccounts()
	if err == nil && len(accounts) > 0 {
		logger.Info("found stored accounts", "accounts", accounts)
		// Load the first account
		if err := client.LoadToken(accounts[0]); err != nil {
			logger.Error("load token", "error", err)
		} else {
			logger.Info("loaded stored token", "account", accounts[0])
		}
	} else {
		// No stored token, need to login
		logger.Info("no stored token, starting login flow...")

		result, err := client.Login(context.Background(), func(ctx context.Context, qr *login.QRCode) error {
			// Display QR code - you can:
			// 1. Print the URL for user to scan
			// 2. Generate QR code image in terminal
			// 3. Show QR code image URL in a web interface
			fmt.Println("\n========================================")
			fmt.Println("Scan this QR code with WeChat to login:")
			fmt.Println(qr.Content)
			if qr.ImageURL != "" {
				fmt.Println("\nOr open this URL:")
				fmt.Println(qr.ImageURL)
			}
			fmt.Println("========================================")
			return nil
		})
		if err != nil {
			logger.Error("login failed", "error", err)
			os.Exit(1)
		}

		logger.Info("login successful",
			"account_id", result.AccountID,
			"user_id", result.UserID,
		)
	}

	// Setup signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		logger.Info("shutting down...")
		cancel()
	}()

	// Run message loop
	logger.Info("starting bot...")
	if err := client.Run(ctx, func(ctx context.Context, msg *ilink.Message) error {
		text := msg.GetText()
		logger.Info("received message",
			"from", msg.FromUserID,
			"text", text,
		)

		// Simple echo response
		reply := "You said: " + text
		return client.SendText(ctx, msg.FromUserID, reply)
	}); err != nil && err != context.Canceled {
		logger.Error("run error", "error", err)
		os.Exit(1)
	}

	logger.Info("bot stopped")
}
