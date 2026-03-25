package main

import (
	"context"
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

	// Create client with token store
	// SDK will automatically load stored token if available
	// SDK also handles login and re-login automatically (default behavior)
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

	// Explicit login (optional - Run() will auto-login if needed)
	// Here we call Login() to show the login result before Run()
	result, err := client.Login(context.Background(), func(ctx context.Context, qr *login.QRCode) error {
		login.PrintQRCodeWithTerm(qr)
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
