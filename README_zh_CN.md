# WeChat iLink SDK for Go

[English](./README.md) | 中文文档

[![Go Reference](https://pkg.go.dev/badge/github.com/the-yex/wechat-ilink-sdk.svg)](https://pkg.go.dev/github.com/the-yex/wechat-ilink-sdk)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

基于 iLink 协议的专业、高度可扩展的微信机器人 Go SDK。

## 功能特性

- **扫码登录** - 扫描二维码认证，Token 本地持久化存储
- **自动重登录** - 自动验证存储的 Token，处理会话过期
- **消息处理** - 收发文本、图片、视频、文件、语音消息
- **CDN 媒体** - AES-128-ECB 加密的媒体文件上传/下载
- **中间件系统** - 内置日志、重试、恢复中间件
- **插件系统** - 可扩展的插件架构，支持自定义功能
- **事件系统** - 异步事件分发，实现松耦合
- **会话管理** - 自动处理上下文 Token 和会话过期
- **生产就绪** - 完善的错误处理、日志记录和测试

## 架构设计

```
Client (Facade)
├── MessageService   → SendText, SendImage, SendTyping
├── MediaService     → Upload, Download
├── AuthService      → Login, SetToken, LoadToken
├── SessionService   → IsPaused, RemainingPause
└── EventDispatcher  → Subscribe, Dispatch
```

SDK 采用分层架构，职责分离清晰：
- **Client** - 统一入口（门面模式）
- **Services** - 领域业务逻辑（Message、Media、Auth、Session）
- **iLink** - 协议层（数据包处理、连接管理）
- **Transport** - 网络抽象

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

    // 创建 Token 存储器（支持自动登录）
    tokenStore, _ := login.NewFileTokenStore("")

    // 定义二维码显示回调
    qrCallback := func(ctx context.Context, qr *login.QRCode) error {
        login.PrintQRCodeWithTerm(qr)
        return nil
    }

    // 创建客户端（配置登录回调）
    client, _ := ilinksdk.NewClient(
        ilinksdk.WithLogger(logger),
        ilinksdk.WithTokenStore(tokenStore),
        ilinksdk.WithOnLogin(qrCallback), // Run() 时自动登录
    )
    defer client.Close()

    // 设置会话过期回调（可选）
    client.SetOnSessionExpired(func(ctx context.Context) (*ilink.LoginResult, error) {
        logger.Info("会话过期，请重新扫码")
        return client.Login(ctx, qrCallback)
    })

    // 直接调用 Run() - SDK 自动处理登录和消息处理
    ctx := context.Background()
    client.Run(ctx, func(ctx context.Context, msg *ilink.Message) error {
        // 自动回复文本消息
        if text := msg.GetText(); text != "" {
            return client.SendText(ctx, msg.FromUserID, "收到: "+text)
        }
        return nil
    })
}
```

## 项目结构

```
wechat-ilink-sdk/
├── client.go              # 主客户端（入口）
├── config.go              # 配置
├── options.go             # 选项模式
├── errors.go              # 错误定义
│
├── types/                 # 核心类型（Message、Requests 等）
├── ilink/                 # API 客户端 & 类型别名
├── login/                 # 登录服务和 Token 存储
├── media/                 # CDN 媒体类型
├── plugin/                # 插件接口
├── middleware/            # 中间件接口
├── event/                 # 事件类型
│
├── internal/              # 内部实现（不对外暴露）
│   ├── service/           # 服务层实现
│   ├── contextmgr/        # 上下文 Token 管理器
│   ├── crypto/            # 加密工具
│   └── httpx/             # HTTP 工具
│
└── examples/              # 示例代码
```

## 配置选项

```go
client, err := ilinksdk.NewClient(
    ilinksdk.WithBaseURL("https://ilinkai.weixin.qq.com"),
    ilinksdk.WithCDNBaseURL("https://novac2c.cdn.weixin.qq.com/c2c"),
    ilinksdk.WithTimeout(30 * time.Second),
    ilinksdk.WithRetry(3, time.Second, 5 * time.Second),
    ilinksdk.WithLogger(slog.Default()),
    ilinksdk.WithTokenStore(tokenStore),
    ilinksdk.WithPlugins(myPlugin1, myPlugin2),
)
```

## Token 管理

### 默认方式（自动管理）

SDK 默认使用文件存储，自动处理 Token 持久化：

```go
// 默认存储在 ./.weixin/default.json
client, _ := ilinksdk.NewClient(
    ilinksdk.WithOnLogin(qrCallback),  // 扫码显示
)

// 或指定存储目录
tokenStore, _ := login.NewFileTokenStore("./my-bot")
client, _ := ilinksdk.NewClient(
    ilinksdk.WithTokenStore(tokenStore),
    ilinksdk.WithOnLogin(qrCallback),
)
```

### 自定义存储（高级）

如果需要自己管理 Token（如存数据库、支持多账号等），使用回调钩子：

```go
client, _ := ilinksdk.NewClient(
    // 登录成功时保存用户信息
    ilinksdk.WithOnLoginSuccess(func(ctx context.Context, result *ilink.LoginResult) error {
        // 保存到数据库
        db.SaveUser(result.AccountID, &User{
            Token:   result.Token,
            BaseURL: result.BaseURL,
            UserID:  result.UserID,
        })
        return nil
    }),

    // 需要加载 Token 时调用
    ilinksdk.WithTokenProvider(func(ctx context.Context) (*login.TokenInfo, error) {
        user := db.GetUser(accountID)
        if user == nil {
            return nil, nil  // 返回 nil 触发登录流程
        }
        return &login.TokenInfo{
            Token:   user.Token,
            BaseURL: user.BaseURL,
            UserID:  user.UserID,
        }, nil
    }),

    // Token 失效时清除
    ilinksdk.WithOnTokenInvalid(func(ctx context.Context) {
        db.DeleteToken(accountID)
    }),
)
```

### 会话过期处理

```go
// 设置会话过期回调
client.SetOnSessionExpired(func(ctx context.Context) (*ilink.LoginResult, error) {
    fmt.Println("会话已过期！请重新扫码登录")
    return client.Login(ctx, func(ctx context.Context, qr *login.QRCode) error {
        login.PrintQRCodeWithTerm(qr)
        return nil
    })
})
```

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

| 事件 | 触发时机 | 数据类型 |
|------|----------|----------|
| `EventTypeMessage` | 收到新消息 | `*ilink.Message` |
| `EventTypeLogin` | 登录成功 | `*ilink.LoginResult` |
| `EventTypeError` | 发生错误 | `error` |
| `EventTypeSessionExpired` | 会话过期 | `nil` |
| `EventTypeConnected` | 客户端启动 | `nil` |
| `EventTypeDisconnected` | 客户端停止 | `nil` |

### 使用事件处理器

```go
client.OnMessage(func(ctx context.Context, e *event.Event) error {
    msg := e.Data.(*ilink.Message)
    log.Printf("收到消息: %s", msg.GetText())
    return nil
})

client.OnSessionExpired(func(ctx context.Context, e *event.Event) error {
    log.Println("会话过期，需要重新登录")
    return nil
})
```

## 插件系统

创建自定义插件扩展 SDK 功能：

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

## CDN 媒体

### 上传媒体文件

```go
result, err := client.UploadMedia(ctx, &media.UploadRequest{
    Data:      imageData,
    MediaType: ilink.UploadMediaTypeImage,
    ToUserID:  "user-id",
})
```

### 下载媒体文件

```go
data, err := client.DownloadMedia(ctx, &media.DownloadRequest{
    EncryptQueryParam: "cdn-param",
    AESKey:            "base64-encoded-key",
})
```

## 错误处理

```go
import "errors"

err := client.SendText(ctx, toUserID, text)
if errors.Is(err, ilinksdk.ErrSessionExpired) {
    // 会话过期，需要重新登录
}
if errors.Is(err, ilinksdk.ErrContextTokenRequired) {
    // 缺少上下文 Token
}
```

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

## 后端 API 协议

SDK 通过 HTTP JSON API 与微信后端通信。主要接口：

| 接口 | 路径 | 说明 |
|------|------|------|
| getUpdates | `getupdates` | 长轮询获取新消息 |
| sendMessage | `sendmessage` | 发送消息 |
| getUploadUrl | `getuploadurl` | 获取 CDN 上传预签名 URL |
| getConfig | `getconfig` | 获取配置（typing ticket 等） |
| sendTyping | `sendtyping` | 发送输入状态 |

### 消息结构

| 字段 | 类型 | 说明 |
|------|------|------|
| `from_user_id` | `string` | 发送者 ID |
| `to_user_id` | `string` | 接收者 ID |
| `message_type` | `number` | `1` = 用户消息, `2` = 机器人消息 |
| `message_state` | `number` | `0` = NEW, `1` = GENERATING, `2` = FINISH |
| `item_list` | `[]MessageItem` | 消息内容列表 |
| `context_token` | `string` | 会话上下文令牌，回复时需回传 |

### MessageItem 类型

| type | 类型 | 说明 |
|------|------|------|
| `1` | TEXT | 文本消息 |
| `2` | IMAGE | 图片消息 |
| `3` | VOICE | 语音消息 |
| `4` | FILE | 文件消息 |
| `5` | VIDEO | 视频消息 |

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