package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	ilinksdk "github.com/the-yex/wechat-ilink-sdk"
	"github.com/the-yex/wechat-ilink-sdk/ilink"
	"github.com/the-yex/wechat-ilink-sdk/login"
)

// ExternalTokenRepository simulates a database or remote credential store.
// Replace it with Redis, SQL, Vault, or your own persistence layer.
type ExternalTokenRepository struct {
	mu    sync.RWMutex
	token *login.TokenInfo
}

func (r *ExternalTokenRepository) Load(ctx context.Context) (*login.TokenInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.token == nil {
		return nil, nil
	}
	copy := *r.token
	return &copy, nil
}

func (r *ExternalTokenRepository) Save(ctx context.Context, token *login.TokenInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	copy := *token
	r.token = &copy
	return nil
}

func (r *ExternalTokenRepository) Delete(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.token = nil
	return nil
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	repo := &ExternalTokenRepository{}

	client, err := ilinksdk.NewClient(
		ilinksdk.WithLogger(logger),
		ilinksdk.WithTokenProvider(func(ctx context.Context) (*login.TokenInfo, error) {
			return repo.Load(ctx)
		}),
		ilinksdk.WithOnLoginSuccess(func(ctx context.Context, result *ilink.LoginResult) error {
			logger.Info("saving token to external repository", "user_id", result.UserID)
			return repo.Save(ctx, &login.TokenInfo{
				Token:   result.Token,
				BaseURL: result.BaseURL,
				UserID:  result.UserID,
			})
		}),
		ilinksdk.WithOnTokenInvalid(func(ctx context.Context) {
			logger.Warn("token became invalid, deleting it from external repository")
			if err := repo.Delete(ctx); err != nil {
				logger.Error("delete token", "error", err)
			}
		}),
	)
	if err != nil {
		logger.Error("create client", "error", err)
		os.Exit(1)
	}
	defer client.Close()

	if current := client.CurrentUser(); current != nil {
		logger.Info("restored token from external repository", "user_id", current.UserID)
	} else {
		logger.Info("no token in external repository yet, QR login will be used on first run")
	}

	client.OnText(func(ctx context.Context, msg *ilink.Message, text string) error {
		logger.Info("received text message", "from", msg.FromUserID, "text", text)
		return client.SendText(ctx, msg.FromUserID, "Echo from external token provider: "+text)
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nShutting down...")
		cancel()
	}()

	if err := client.Run(ctx, nil); err != nil && err != context.Canceled {
		logger.Error("run error", "error", err)
		os.Exit(1)
	}
}
