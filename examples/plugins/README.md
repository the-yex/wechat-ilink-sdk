# 插件开发指南

本目录展示了如何为 wechat-ilink-sdk 开发和自定义插件。

## 快速开始

### 运行示例

```bash
go run ./examples/plugins/main.go
```

## 插件接口

插件需要实现 `plugin.Plugin` 接口：

```go
type Plugin interface {
    // Name 返回插件名称
    Name() string

    // Initialize 在插件注册时调用
    Initialize(ctx context.Context, sdk SDK) error

    // OnMessage 在收到每条消息时调用
    OnMessage(ctx context.Context, msg *ilink.Message) error

    // OnError 在发生错误时调用
    OnError(ctx context.Context, err error)
}
```

### 接口说明

| 方法 | 调用时机 | 说明 |
|------|----------|------|
| `Name()` | 注册时 | 返回唯一的插件名称 |
| `Initialize()` | 注册时 | 初始化插件，可访问 SDK 功能 |
| `OnMessage()` | 每条消息 | 处理消息，返回 error 停止后续处理 |
| `OnError()` | 发生错误 | 错误通知，不返回值 |

## SDK 接口

插件通过 `SDK` 接口访问 SDK 功能：

```go
type SDK interface {
    SendMessage(ctx context.Context, req *ilink.SendMessageRequest) error
    SendText(ctx context.Context, toUserID, text string) error
    UploadMedia(ctx context.Context, req *media.UploadRequest) (*media.UploadResult, error)
    DownloadMedia(ctx context.Context, req *media.DownloadRequest) ([]byte, error)
}
```

## 示例插件

### 1. 日志记录插件

```go
type LoggerPlugin struct {
    logger *slog.Logger
    prefix string
}

func (p *LoggerPlugin) Name() string {
    return p.prefix + "-logger"
}

func (p *LoggerPlugin) Initialize(ctx context.Context, sdk plugin.SDK) error {
    p.logger.Info("插件已初始化", "name", p.Name())
    return nil
}

func (p *LoggerPlugin) OnMessage(ctx context.Context, msg *ilink.Message) error {
    p.logger.Debug("收到消息",
        "from", msg.FromUserID,
        "text", msg.GetText(),
    )
    return nil // 返回 nil 继续其他插件处理
}

func (p *LoggerPlugin) OnError(ctx context.Context, err error) {
    p.logger.Error("插件捕获错误", "name", p.Name(), "error", err)
}
```

### 2. 命令处理插件

```go
type CommandPlugin struct {
    commands map[string]CommandHandler
    sdk      plugin.SDK
}

func (p *CommandPlugin) Initialize(ctx context.Context, sdk plugin.SDK) error {
    p.sdk = sdk
    p.Register("help", p.handleHelp)
    p.Register("ping", p.handlePing)
    return nil
}

func (p *CommandPlugin) OnMessage(ctx context.Context, msg *ilink.Message) error {
    text := msg.GetText()
    if strings.HasPrefix(text, "/") {
        return p.processCommand(ctx, msg.FromUserID, text)
    }
    return nil
}

func (p *CommandPlugin) handleHelp(ctx context.Context, fromUserID string, args []string) error {
    return p.sdk.SendText(ctx, fromUserID, "可用命令：/help, /ping, /echo")
}
```

### 3. 关键词自动回复插件

```go
type AutoReplyPlugin struct {
    replies map[string]string
    sdk     plugin.SDK
}

func (p *AutoReplyPlugin) Initialize(ctx context.Context, sdk plugin.SDK) error {
    p.sdk = sdk
    p.AddReply("你好", "你好！有什么可以帮助你的吗？")
    p.AddReply("hello", "Hello! How can I help you?")
    return nil
}

func (p *AutoReplyPlugin) OnMessage(ctx context.Context, msg *ilink.Message) error {
    text := msg.GetText()
    for keyword, reply := range p.replies {
        if strings.Contains(text, keyword) {
            return p.sdk.SendText(ctx, msg.FromUserID, reply)
        }
    }
    return nil
}
```

## 注册和使用插件

### 方法 1: 使用 UsePlugin

```go
client, err := ilinksdk.NewClient(...)

// 创建插件
loggerPlugin := NewLoggerPlugin(logger, "app")
commandPlugin := NewCommandPlugin()

// 注册插件
if err := client.UsePlugin(loggerPlugin); err != nil {
    log.Fatal(err)
}
if err := client.UsePlugin(commandPlugin); err != nil {
    log.Fatal(err)
}
```

### 方法 2: 使用 WithPlugins

```go
client, err := ilinksdk.NewClient(
    ilinksdk.WithLogger(logger),
    ilinksdk.WithPlugins(
        NewLoggerPlugin(logger, "app"),
        NewCommandPlugin(),
        NewAutoReplyPlugin(),
    ),
)
```

## 插件执行顺序

1. 消息到达时，按注册顺序依次调用每个插件的 `OnMessage()`
2. 如果插件返回 `nil`，继续下一个插件
3. 如果插件返回 `error`，停止后续处理
4. 所有插件处理完成后，调用主消息处理器

## 最佳实践

### 1. 错误处理

```go
func (p *MyPlugin) OnMessage(ctx context.Context, msg *ilink.Message) error {
    if err := p.process(msg); err != nil {
        // 返回错误停止后续处理
        return fmt.Errorf("process message: %w", err)
    }
    return nil
}
```

### 2. 上下文使用

```go
func (p *MyPlugin) OnMessage(ctx context.Context, msg *ilink.Message) error {
    // 使用传入的上下文
    if err := ctx.Err(); err != nil {
        return err // 上下文已取消
    }

    // 创建带超时的子上下文
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    return p.doWork(ctx, msg)
}
```

### 3. 并发安全

```go
type SafePlugin struct {
    mu       sync.RWMutex
    data     map[string]string
}

func (p *SafePlugin) OnMessage(ctx context.Context, msg *ilink.Message) error {
    p.mu.RLock()
    defer p.mu.RUnlock()
    // 安全读取数据
    return nil
}
```

### 4. 避免阻塞

```go
// 不好的做法 - 阻塞主流程
func (p *MyPlugin) OnMessage(ctx context.Context, msg *ilink.Message) error {
    time.Sleep(10 * time.Second) // 阻塞！
    return nil
}

// 好的做法 - 异步处理
func (p *MyPlugin) OnMessage(ctx context.Context, msg *ilink.Message) error {
    go func() {
        time.Sleep(10 * time.Second)
        // 异步完成工作
    }()
    return nil // 立即返回
}
```

## 调试技巧

### 1. 日志输出

```go
type DebugPlugin struct {
    logger *slog.Logger
}

func (p *DebugPlugin) OnMessage(ctx context.Context, msg *ilink.Message) error {
    p.logger.Debug("插件处理消息",
        "plugin", p.Name(),
        "from", msg.FromUserID,
        "text", msg.GetText(),
    )
    return nil
}
```

### 2. 错误收集

```go
type ErrorTrackingPlugin struct {
    errors []error
    mu     sync.Mutex
}

func (p *ErrorTrackingPlugin) OnError(ctx context.Context, err error) {
    p.mu.Lock()
    defer p.mu.Unlock()
    p.errors = append(p.errors, err)
}
```

## 完整示例

查看 `examples/plugins/main.go` 获取完整的可运行示例，包含：
- 日志记录插件
- 命令处理插件（/help, /ping, /echo）
- 关键词自动回复插件

## 常见问题

### Q: 如何停止其他插件处理？

A: 在 `OnMessage()` 中返回错误：

```go
func (p *MyPlugin) OnMessage(ctx context.Context, msg *ilink.Message) error {
    if p.shouldStop(msg) {
        return fmt.Errorf("stop processing")
    }
    return nil // 继续其他插件
}
```

### Q: 插件可以发送消息吗？

A: 可以，通过 SDK 接口：

```go
func (p *MyPlugin) Initialize(ctx context.Context, sdk plugin.SDK) error {
    p.sdk = sdk // 保存 SDK 引用
    return nil
}

func (p *MyPlugin) OnMessage(ctx context.Context, msg *ilink.Message) error {
    return p.sdk.SendText(ctx, msg.FromUserID, "自动回复")
}
```

### Q: 如何调试插件？

A: 启用调试日志：

```go
logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))
```
