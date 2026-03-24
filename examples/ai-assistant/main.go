package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/the-yex/wechat-ilink-sdk"
	"github.com/the-yex/wechat-ilink-sdk/ilink"
	"github.com/the-yex/wechat-ilink-sdk/middleware"
)

// MockLLMClient represents an LLM client interface.
// Replace with your actual LLM client (OpenAI, Anthropic, etc.)
type MockLLMClient struct{}

func (c *MockLLMClient) Chat(ctx context.Context, prompt string) (string, error) {
	// Replace with actual LLM API call
	return "AI Response: " + prompt, nil
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	token := os.Getenv("WEIXIN_TOKEN")
	if token == "" {
		logger.Error("WEIXIN_TOKEN environment variable is required")
		os.Exit(1)
	}

	client, err := ilinksdk.NewClient(
		ilinksdk.WithToken(token),
		ilinksdk.WithLogger(logger),
		ilinksdk.WithMiddleware(
			middleware.Logging(logger),
			middleware.Retry(middleware.RetryConfig{
				MaxAttempts: 3,
				WaitMin:     1e9,  // 1 second
				WaitMax:     5e9,  // 5 seconds
			}),
		),
	)
	if err != nil {
		logger.Error("create client", "error", err)
		os.Exit(1)
	}
	defer client.Close()

	// Initialize LLM client
	llm := &MockLLMClient{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		cancel()
	}()

	logger.Info("starting AI assistant bot...")

	if err := client.Run(ctx, func(ctx context.Context, msg *ilink.Message) error {
		text := msg.GetText()
		if text == "" {
			return nil
		}

		logger.Info("processing message", "from", msg.FromUserID, "text", text)

		// Send typing indicator
		if err := client.SendTyping(ctx, msg.FromUserID, true); err != nil {
			logger.Warn("send typing", "error", err)
		}

		// Call LLM
		reply, err := llm.Chat(ctx, text)
		if err != nil {
			logger.Error("llm error", "error", err)
			return client.SendText(ctx, msg.FromUserID, "Sorry, I encountered an error processing your request.")
		}

		// Cancel typing
		_ = client.SendTyping(ctx, msg.FromUserID, false)

		// Send reply
		return client.SendText(ctx, msg.FromUserID, reply)
	}); err != nil && err != context.Canceled {
		logger.Error("run error", "error", err)
		os.Exit(1)
	}

	logger.Info("bot stopped")
}