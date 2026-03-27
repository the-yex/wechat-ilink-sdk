package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}

	// Create client with token store and production-ready defaults.
	// SDK will automatically load stored token if available and perform QR login when needed.
	client, err := ilinksdk.NewClient(
		ilinksdk.WithLogger(logger),
		ilinksdk.WithTokenStore(tokenStore),
		ilinksdk.WithHTTPClient(&http.Client{
			Timeout:   15 * time.Second,
			Transport: transport,
		}),
		ilinksdk.WithLongPollHTTPClient(&http.Client{
			Timeout:   40 * time.Second,
			Transport: transport,
		}),
		ilinksdk.WithCDNHTTPClient(&http.Client{
			Transport: transport,
		}),
		ilinksdk.WithRetry(3, time.Second, 5*time.Second),
		ilinksdk.WithRateLimit(5, 1),
		ilinksdk.WithMiddleware(
			middleware.Logging(logger),
			middleware.Recovery(logger),
		),
	)
	if err != nil {
		logger.Error("create client", "error", err)
		os.Exit(1)
	}
	defer client.Close()

	client.OnText(func(ctx context.Context, msg *ilink.Message, text string) error {
		logger.Info("received text message",
			"from", msg.FromUserID,
			"text", text,
		)

		return client.SendText(ctx, msg.FromUserID, "You said: "+text)
	})

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

	// Run message loop with registered handlers.
	logger.Info("starting bot...")
	if err := client.Run(ctx, nil); err != nil && err != context.Canceled {
		logger.Error("run error", "error", err)
		os.Exit(1)
	}

	logger.Info("bot stopped")
}
