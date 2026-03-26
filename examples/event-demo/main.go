package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	ilinksdk "github.com/the-yex/wechat-ilink-sdk"
	"github.com/the-yex/wechat-ilink-sdk/event"
	"github.com/the-yex/wechat-ilink-sdk/ilink"
	"github.com/the-yex/wechat-ilink-sdk/login"
	"github.com/the-yex/wechat-ilink-sdk/plugin"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n正在关闭...")
		cancel()
	}()

	// Setup logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Create token store
	tokenStore, err := login.NewFileTokenStore("")
	if err != nil {
		logger.Error("创建 Token 存储失败", "error", err)
		os.Exit(1)
	}

	// Create client
	client, err := ilinksdk.NewClient(
		ilinksdk.WithLogger(logger),
		ilinksdk.WithTokenStore(tokenStore),
		ilinksdk.WithPlugins(plugin.NewLogoutPlugin(nil)),
	)
	if err != nil {
		logger.Error("创建客户端失败", "error", err)
		os.Exit(1)
	}
	defer client.Close()

	// ========================================
	// 事件订阅示例
	// ========================================

	// 1. 登录成功事件 - 记录用户信息
	client.Events().Subscribe(event.EventTypeLogin, func(ctx context.Context, e *event.Event) error {
		result := e.Data.(*ilink.LoginResult)
		logger.Info("🎉 登录成功",
			"user_id", result.UserID,
			"account_id", result.AccountID,
		)
		// 这里可以初始化业务状态、发送通知等
		return nil
	})

	// 2. 会话过期事件 - 清理状态、发送告警
	client.Events().Subscribe(event.EventTypeSessionExpired, func(ctx context.Context, e *event.Event) error {
		logger.Warn("⚠️ 会话已过期，SDK 将自动重新登录")
		// 这里可以清理本地缓存、发送告警通知等
		return nil
	})

	// 3. 连接成功事件 - 标记服务状态
	client.Events().Subscribe(event.EventTypeConnected, func(ctx context.Context, e *event.Event) error {
		logger.Info("✅ 已连接到服务器，开始接收消息")
		// 这里可以更新服务状态为"在线"
		return nil
	})

	// 4. 断开连接事件 - 标记服务离线
	client.Events().Subscribe(event.EventTypeDisconnected, func(ctx context.Context, e *event.Event) error {
		logger.Info("❌ 已断开连接")
		// 这里可以更新服务状态为"离线"
		return nil
	})

	// 5. 错误事件 - 统一错误处理、监控上报
	client.Events().Subscribe(event.EventTypeError, func(ctx context.Context, e *event.Event) error {
		err := e.Data.(error)
		logger.Error("发生错误", "error", err)
		// 这里可以上报到 Sentry、Prometheus 等监控系统
		return nil
	})

	// ========================================
	// 消息处理
	// ========================================

	client.OnText(func(ctx context.Context, msg *ilink.Message, text string) error {
		logger.Info("收到消息", "from", msg.FromUserID, "text", text)

		return client.SendText(ctx, msg.FromUserID, "收到: "+text)
	})

	// ========================================
	// 定时发送消息（演示主动发送功能）
	// ========================================
	stopTicker := make(chan struct{})
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:

				if err := client.SendText(ctx, client.CurrentUser().UserID, "你好"); err != nil {
					logger.Error("发送消息失败", "error", err)
				}

			case <-stopTicker:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
	defer close(stopTicker)

	// Run the bot
	fmt.Println("启动机器人...")
	if err := client.Run(ctx, nil); err != nil && err != context.Canceled {
		logger.Error("运行错误", "error", err)
	}

	fmt.Println("已关闭")
}
