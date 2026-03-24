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
	"github.com/the-yex/wechat-ilink-sdk/plugin"
)

// ---------------------------------------------------------------------------
// Plugin 示例 1: 日志记录插件
// ---------------------------------------------------------------------------

// LoggerPlugin 记录所有消息和错误的插件
type LoggerPlugin struct {
	logger *slog.Logger
	prefix string
}

// NewLoggerPlugin 创建日志记录插件
func NewLoggerPlugin(logger *slog.Logger, prefix string) *LoggerPlugin {
	return &LoggerPlugin{
		logger: logger,
		prefix: prefix,
	}
}

// Name 返回插件名称
func (p *LoggerPlugin) Name() string {
	return p.prefix + "-logger"
}

// Initialize 在插件注册时调用
func (p *LoggerPlugin) Initialize(ctx context.Context, sdk plugin.SDK) error {
	p.logger.Info("插件已初始化", "name", p.Name())
	return nil
}

// OnMessage 在收到每条消息时调用
func (p *LoggerPlugin) OnMessage(ctx context.Context, msg *ilink.Message) error {
	p.logger.Debug("收到消息",
		"from", msg.FromUserID,
		"to", msg.ToUserID,
		"text", msg.GetText(),
	)
	return nil // 返回 nil 继续处理
}

// OnError 在发生错误时调用
func (p *LoggerPlugin) OnError(ctx context.Context, err error) {
	p.logger.Error("插件捕获错误", "name", p.Name(), "error", err)
}

// ---------------------------------------------------------------------------
// Plugin 示例 2: 命令处理器插件
// ---------------------------------------------------------------------------

// CommandPlugin 处理特定命令的插件
type CommandPlugin struct {
	commands map[string]CommandHandler
	sdk      plugin.SDK
}

// CommandHandler 是命令处理函数类型
type CommandHandler func(ctx context.Context, fromUserID string, args []string) error

// NewCommandPlugin 创建命令处理插件
func NewCommandPlugin() *CommandPlugin {
	return &CommandPlugin{
		commands: make(map[string]CommandHandler),
	}
}

// Name 返回插件名称
func (p *CommandPlugin) Name() string {
	return "command-processor"
}

// Initialize 初始化插件并注册命令
func (p *CommandPlugin) Initialize(ctx context.Context, sdk plugin.SDK) error {
	p.sdk = sdk

	// 注册命令
	p.Register("help", p.handleHelp)
	p.Register("ping", p.handlePing)
	p.Register("echo", p.handleEcho)

	fmt.Println("命令处理插件已初始化，支持的命令：help, ping, echo")
	return nil
}

// Register 注册一个命令
func (p *CommandPlugin) Register(cmd string, handler CommandHandler) {
	p.commands[cmd] = handler
}

// OnMessage 处理消息中的命令
func (p *CommandPlugin) OnMessage(ctx context.Context, msg *ilink.Message) error {
	text := msg.GetText()
	if text == "" {
		return nil
	}

	// 检查是否以 / 开头的命令
	if len(text) > 0 && text[0] == '/' {
		return p.processCommand(ctx, msg.FromUserID, text)
	}

	return nil // 不是命令，继续其他插件处理
}

// processCommand 解析并执行命令
func (p *CommandPlugin) processCommand(ctx context.Context, fromUserID string, text string) error {
	// 解析命令：/cmd arg1 arg2 ...
	cmd, args, _ := parseCommand(text)

	if handler, ok := p.commands[cmd]; ok {
		return handler(ctx, fromUserID, args)
	}

	// 未知命令
	return p.sdk.SendText(ctx, fromUserID, fmt.Sprintf("未知命令：%s\n使用 /help 查看帮助", cmd))
}

// handleHelp 处理 help 命令
func (p *CommandPlugin) handleHelp(ctx context.Context, fromUserID string, args []string) error {
	helpText := `可用命令：
/help - 显示帮助信息
/ping - 测试机器人响应
/echo <消息> - 回显消息`
	return p.sdk.SendText(ctx, fromUserID, helpText)
}

// handlePing 处理 ping 命令
func (p *CommandPlugin) handlePing(ctx context.Context, fromUserID string, args []string) error {
	return p.sdk.SendText(ctx, fromUserID, "pong! 🏓")
}

// handleEcho 处理 echo 命令
func (p *CommandPlugin) handleEcho(ctx context.Context, fromUserID string, args []string) error {
	if len(args) == 0 {
		return p.sdk.SendText(ctx, fromUserID, "请提供要回显的消息：/echo <消息>")
	}
	// 拼接所有参数
	message := joinStrings(args)
	return p.sdk.SendText(ctx, fromUserID, message)
}

// OnError 处理错误
func (p *CommandPlugin) OnError(ctx context.Context, err error) {
	fmt.Printf("命令插件错误：%v\n", err)
}

// parseCommand 解析命令字符串
func parseCommand(text string) (cmd string, args []string, ok bool) {
	if len(text) == 0 || text[0] != '/' {
		return "", nil, false
	}

	// 去掉 / 前缀
	text = text[1:]

	// 分割命令和参数
	cmd = ""
	args = []string{}
	currentArg := ""

	for i := 0; i < len(text); i++ {
		c := text[i]
		if c == ' ' {
			if cmd == "" {
				cmd = currentArg
				currentArg = ""
			} else if currentArg != "" {
				args = append(args, currentArg)
				currentArg = ""
			}
		} else {
			currentArg += string(c)
		}
	}

	// 处理最后一个部分
	if cmd == "" {
		cmd = currentArg
	} else if currentArg != "" {
		args = append(args, currentArg)
	}

	return cmd, args, true
}

// joinStrings 拼接字符串数组
func joinStrings(strs []string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += " "
		}
		result += s
	}
	return result
}

// ---------------------------------------------------------------------------
// Plugin 示例 3: 关键词自动回复插件
// ---------------------------------------------------------------------------

// AutoReplyPlugin 关键词自动回复插件
type AutoReplyPlugin struct {
	replies map[string]string
}

// NewAutoReplyPlugin 创建自动回复插件
func NewAutoReplyPlugin() *AutoReplyPlugin {
	return &AutoReplyPlugin{
		replies: make(map[string]string),
	}
}

// Name 返回插件名称
func (p *AutoReplyPlugin) Name() string {
	return "auto-reply"
}

// Initialize 初始化并注册关键词回复
func (p *AutoReplyPlugin) Initialize(ctx context.Context, sdk plugin.SDK) error {
	// 注册关键词回复
	p.AddReply("你好", "你好！有什么可以帮助你的吗？")
	p.AddReply("hello", "Hello! How can I help you?")
	p.AddReply("谢谢", "不客气！")
	p.AddReply("再见", "再见，祝你有美好的一天！")
	p.AddReply("你是谁", "我是一个微信机器人助手。")

	fmt.Println("自动回复插件已初始化")
	return nil
}

// AddReply 添加关键词回复
func (p *AutoReplyPlugin) AddReply(keyword, reply string) {
	p.replies[keyword] = reply
}

// OnMessage 检查并回复匹配的消息
func (p *AutoReplyPlugin) OnMessage(ctx context.Context, msg *ilink.Message) error {
	text := msg.GetText()
	if text == "" {
		return nil
	}

	// 检查是否有关键词匹配
	for keyword, reply := range p.replies {
		if contains(text, keyword) {
			return sendText(ctx, p, msg.FromUserID, reply)
		}
	}

	return nil // 没有匹配，继续其他插件处理
}

// OnError 处理错误
func (p *AutoReplyPlugin) OnError(ctx context.Context, err error) {
	fmt.Printf("自动回复插件错误：%v\n", err)
}

// contains 检查字符串是否包含子串（不区分大小写）
func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsIgnoreCase(s, substr)
}

func containsIgnoreCase(s, substr string) bool {
	s = toLower(s)
	substr = toLower(substr)
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c = c + 32
		}
		result[i] = c
	}
	return string(result)
}

// sendText 辅助函数，用于发送消息
func sendText(ctx context.Context, p *AutoReplyPlugin, toUserID, text string) error {
	// 这里需要通过 SDK 发送，但为了简化示例，直接打印
	fmt.Printf("[自动回复] %s: %s\n", toUserID, text)
	return nil
}

// ---------------------------------------------------------------------------
// Main - 演示插件使用
// ---------------------------------------------------------------------------

func main() {
	// Setup logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	fmt.Println("=== WeChat iLink SDK - Plugin 插件示例 ===")

	// 创建文件 Token 存储
	tokenStore, err := login.NewFileTokenStore("")
	if err != nil {
		logger.Error("创建 Token 存储失败", "error", err)
		os.Exit(1)
	}

	// 创建插件
	loggerPlugin := NewLoggerPlugin(logger, "app")
	commandPlugin := NewCommandPlugin()
	autoReplyPlugin := NewAutoReplyPlugin()

	// 创建客户端并注册插件
	client, err := ilinksdk.NewClient(
		ilinksdk.WithLogger(logger),
		ilinksdk.WithTokenStore(tokenStore),
	)
	if err != nil {
		logger.Error("创建客户端失败", "error", err)
		os.Exit(1)
	}
	defer client.Close()

	// 检查是否有存储的 Token
	accounts, err := client.ListAccounts()
	if err == nil && len(accounts) > 0 {
		fmt.Printf("找到已存储的账户：%s\n", accounts[0])
		client.LoadToken(accounts[0])
	} else {
		// 扫码登录
		fmt.Println("未找到存储的 Token，开始扫码登录...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		result, err := client.Login(ctx, func(ctx context.Context, qr *login.QRCode) error {
			login.PrintQRCodeWithTerm(qr)
			return nil
		})
		if err != nil {
			fmt.Printf("登录失败：%v\n", err)
			os.Exit(1)
		}
		fmt.Printf("登录成功！账户：%s\n", result.AccountID)
	}

	// 注册插件到客户端
	if err := client.UsePlugin(context.Background(), loggerPlugin); err != nil {
		logger.Error("注册日志插件失败", "error", err)
	}
	if err := client.UsePlugin(context.Background(), commandPlugin); err != nil {
		logger.Error("注册命令插件失败", "error", err)
	}
	if err := client.UsePlugin(context.Background(), autoReplyPlugin); err != nil {
		logger.Error("注册自动回复插件失败", "error", err)
	}

	fmt.Println("\n已注册以下插件:")
	fmt.Println("  - logger: 消息日志记录")
	fmt.Println("  - command-processor: 命令处理 (/help, /ping, /echo)")
	fmt.Println("  - auto-reply: 关键词自动回复")
	fmt.Println("\n按 Ctrl+C 退出")

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		<-sigChan
		fmt.Println("\n正在关闭...")
		cancel()
	}()

	// 运行消息循环
	if err := client.Run(ctx, func(ctx context.Context, msg *ilink.Message) error {
		// 主处理器 - 插件会先处理消息
		text := msg.GetText()
		logger.Info("主处理器收到消息", "text", text)
		return nil
	}); err != nil && err != context.Canceled {
		logger.Error("运行错误", "error", err)
	}

	fmt.Println("机器人已停止")
}
