# WeChat iLink SDK for Go

[English](./README.md) | 中文文档

[![Go Reference](https://pkg.go.dev/badge/github.com/the-yex/wechat-ilink-sdk.svg)](https://pkg.go.dev/github.com/the-yex/wechat-ilink-sdk)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

基于 iLink 协议的专业、高度可扩展的微信机器人 Go SDK。

## 功能特性

- **扫码登录** - 扫描二维码认证，Token 本地持久化存储
- **自动重登录** - 自动验证存储的 Token，处理会话过期
- **消息处理** - 收发文本、图片、视频、文件、语音消息
- **中间件系统** - 内置日志、重试、恢复中间件
- **插件系统** - 可扩展的插件架构，支持自定义功能
- **事件系统** - 异步事件分发，实现松耦合
- **生产就绪** - 完善的错误处理、日志记录和测试

## 安装

```bash
go get github.com/the-yex/wechat-ilink-sdk
```

## 快速开始

```go
package main

import (
    "context"
    "log/slog"
    "os"

    "github.com/the-yex/wechat-ilink-sdk"
    "github.com/the-yex/wechat-ilink-sdk/ilink"
    "github.com/the-yex/wechat-ilink-sdk/login"
)

func main() {
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

    // 创建 Token 存储器
    tokenStore, _ := login.NewFileTokenStore("")

    // 创建客户端
    client, _ := ilinksdk.NewClient(
        ilinksdk.WithLogger(logger),
        ilinksdk.WithTokenStore(tokenStore),
    )
    defer client.Close()

    // 运行机器人 - SDK 自动处理登录和消息
    ctx := context.Background()
    client.Run(ctx, func(ctx context.Context, msg *ilink.Message) error {
        if text := msg.GetText(); text != "" {
            return client.SendText(ctx, msg.FromUserID, "收到: "+text)
        }
        return nil
    })
}
```

## 交互流程

### 登录流程

```
┌─────────────┐     获取二维码      ┌─────────────┐
│   用户      │ ◄───────────────── │  iLink 服务  │
│             │                    │             │
│   扫码确认   │ ─────────────────► │             │
└─────────────┘                    └─────────────┘
       │
       ▼
  获取 bot_token
  自动保存到 TokenStore
```

SDK 自动处理：显示二维码 → 轮询扫码状态 → 保存凭证。

### 消息流程

```
┌──────────┐                    ┌──────────┐
│ 微信用户  │ ──发送消息────────► │ iLink 服务 │
└──────────┘                    └────┬─────┘
                                     │
                                     ▼
┌──────────┐                    ┌──────────┐
│  Bot SDK │ ◄──长轮询获取消息─── │ iLink 服务 │
│          │ ────发送回复───────► │          │
└──────────┘                    └──────────┘
       │
       ▼
  用户收到回复
```

**关键概念**：
- SDK 通过长轮询接收消息，不是 WebSocket
- 每条消息绑定的 `context_token` 会自动管理，无需手动处理

### 消息类型

| type | 类型 | 说明 |
|------|------|------|
| `1` | TEXT | 文本消息 |
| `2` | IMAGE | 图片消息 |
| `3` | VOICE | 语音消息 |
| `4` | FILE | 文件消息 |
| `5` | VIDEO | 视频消息 |

## 发送消息

```go
// 发送文本
client.SendText(ctx, toUserID, "Hello!")

// 发送图片
client.SendImage(ctx, toUserID, imageData)

// 发送视频
client.SendVideo(ctx, toUserID, videoData)

// 发送文件
client.SendFile(ctx, toUserID, "document.pdf", fileData)

// 发送输入状态
client.SendTyping(ctx, toUserID, true)  // 开始输入
client.SendTyping(ctx, toUserID, false) // 停止输入
```

## 接收消息

```go
client.Run(ctx, func(ctx context.Context, msg *ilink.Message) error {
    // 判断消息类型
    if !msg.IsFromUser() {
        return nil // 忽略非用户消息
    }

    // 处理文本消息
    if text := msg.GetText(); text != "" {
        fmt.Printf("收到文本: %s\n", text)
    }

    // 处理图片消息
    if item := msg.GetFirstMediaItem(); item != nil {
        switch item.Type {
        case types.MessageItemTypeImage:
            // 下载图片
            data, _ := client.DownloadMedia(ctx, &media.DownloadRequest{
                EncryptQueryParam: item.ImageItem.Media.EncryptQueryParam,
                AESKey:            item.ImageItem.Media.AESKey,
            })
            // 回复图片
            client.SendImage(ctx, msg.FromUserID, data)
        }
    }

    return nil
})
```

## 配置选项

```go
client, _ := ilinksdk.NewClient(
    ilinksdk.WithLogger(logger),
    ilinksdk.WithTokenStore(tokenStore),
    ilinksdk.WithMiddleware(
        middleware.Logging(logger),
        middleware.Recovery(logger),
        middleware.Retry(middleware.DefaultRetryConfig()),
    ),
)
```

## Token 管理

SDK 默认使用文件存储 Token，自动处理持久化：

```go
// 默认存储在 ./.weixin/default.json
tokenStore, _ := login.NewFileTokenStore("")

// 或指定存储目录
tokenStore, _ := login.NewFileTokenStore("./my-bot")
```

### 自定义存储

实现 `TokenStore` 接口即可自定义存储方式：

```go
type TokenStore interface {
    Get(ctx context.Context) (*TokenInfo, error)
    Set(ctx context.Context, info *TokenInfo) error
    Clear(ctx context.Context) error
}
```

SQLite 存储示例请参考 [examples/sqlite-storage](./examples/sqlite-storage/)。

## 中间件

### 内置中间件

| 中间件 | 描述 |
|--------|------|
| `Logging` | 请求/响应日志 |
| `Retry` | 指数退避自动重试 |
| `Recovery` | Panic 恢复 |

### 自定义中间件

```go
func CustomMiddleware() middleware.Middleware {
    return func(next middleware.Handler) middleware.Handler {
        return func(ctx context.Context, req *ilink.SendMessageRequest) error {
            // 前置处理
            err := next(ctx, req)
            // 后置处理
            return err
        }
    }
}

client.Use(CustomMiddleware())
```

## 事件系统

### 可用事件

| 事件 | 触发时机 |
|------|----------|
| `EventTypeMessage` | 收到新消息 |
| `EventTypeLogin` | 登录成功 |
| `EventTypeError` | 发生错误 |
| `EventTypeSessionExpired` | 会话过期 |
| `EventTypeConnected` | 客户端启动 |
| `EventTypeDisconnected` | 客户端停止 |

### 使用事件

```go
client.OnMessage(func(ctx context.Context, e *event.Event) error {
    msg := e.Data.(*ilink.Message)
    log.Printf("收到消息: %s", msg.GetText())
    return nil
})
```

## 插件系统

```go
type MyPlugin struct{}

func (p *MyPlugin) Name() string { return "my-plugin" }
func (p *MyPlugin) Initialize(ctx context.Context, sdk plugin.SDK) error { return nil }
func (p *MyPlugin) OnMessage(ctx context.Context, msg *ilink.Message) error {
    // 处理消息
    return nil
}
func (p *MyPlugin) OnError(ctx context.Context, err error) {}

// 注册插件
client.UsePlugin(context.Background(), &MyPlugin{})
```

详细插件开发指南请参考 [examples/plugins/README.md](./examples/plugins/README.md)。

## 示例

查看 [examples](./examples/) 目录：

| 示例 | 描述 |
|------|------|
| `simple-login` | 基础扫码登录 |
| `qrcode-login` | 登录 + Token 存储 |
| `qrcode-login-with-image` | 完整机器人 + 自动回复 |
| `auto-relogin` | 会话过期自动重登录 |
| `sqlite-storage` | SQLite 存储用户信息 |
| `basic-bot` | Echo 机器人 + 中间件 |
| `plugins` | 插件开发示例 |
| `ai-assistant` | AI 助手集成模式 |

## 开发

```bash
# 运行测试
go test ./...

# 测试覆盖率
go test -cover ./...

# 代码检查
golangci-lint run
```

## 许可证

MIT License - 详见 [LICENSE](LICENSE)