package plugin

import (
	"context"
	"fmt"

	"github.com/the-yex/wechat-ilink-sdk/ilink"
)

// LogoutPlugin provides exit command functionality.
// Usage: /exit - logs out the current user and clears stored token.
type LogoutPlugin struct {
	sdk       SDK
	onExit    func(ctx context.Context) error // callback after exit
	enabled   bool
	confirmed bool
}

// NewLogoutPlugin creates a new exit plugin with an optional callback.
// The callback is called after successful logout.
// If no callback is needed, use NewLogoutPlugin(nil) or just NewLogoutPlugin().
func NewLogoutPlugin(onExit ...func(ctx context.Context) error) *LogoutPlugin {
	var callback func(ctx context.Context) error
	if len(onExit) > 0 && onExit[0] != nil {
		callback = onExit[0]
	}
	return &LogoutPlugin{
		onExit:  callback,
		enabled: true,
	}
}

// Name returns the plugin name.
func (p *LogoutPlugin) Name() string {
	return "logout"
}

// Initialize initializes the plugin.
func (p *LogoutPlugin) Initialize(ctx context.Context, sdk SDK) error {
	p.sdk = sdk
	return nil
}

// OnMessage handles exit commands.
func (p *LogoutPlugin) OnMessage(ctx context.Context, msg *ilink.Message) error {
	text := msg.GetText()
	if text == "" {
		return nil
	}

	// Handle /exit command
	if text == "/exit" {
		if !p.enabled {
			return p.sdk.SendText(ctx, msg.FromUserID, "退出功能已禁用")
		}

		if !p.confirmed {
			p.confirmed = true
			return p.sdk.SendText(ctx, msg.FromUserID, "确定要退出吗？再次发送 /exit 确认退出。")
		}

		// Perform exit
		return p.performExit(ctx, msg.FromUserID)
	}

	// Reset confirmation on any other message
	if p.confirmed && text != "" {
		p.confirmed = false
	}

	return nil
}

// OnError handles errors.
func (p *LogoutPlugin) OnError(ctx context.Context, err error) {
	fmt.Printf("[LogoutPlugin] Error: %v\n", err)
}

// performExit executes the exit process.
func (p *LogoutPlugin) performExit(ctx context.Context, fromUserID string) error {
	// Send exit notification
	_ = p.sdk.SendText(ctx, fromUserID, "正在退出，请重新扫码登录...")

	// Call SDK Logout to clear token and trigger re-login
	if err := p.sdk.Logout(ctx); err != nil {
		return err
	}

	// Call onExit callback if set
	if p.onExit != nil {
		if err := p.onExit(ctx); err != nil {
			fmt.Printf("[LogoutPlugin] onExit callback error: %v\n", err)
		}
	}

	return nil
}

// Enable enables the exit functionality.
func (p *LogoutPlugin) Enable() {
	p.enabled = true
}

// Disable disables the exit functionality.
func (p *LogoutPlugin) Disable() {
	p.enabled = false
}

// IsEnabled returns whether exit is enabled.
func (p *LogoutPlugin) IsEnabled() bool {
	return p.enabled
}