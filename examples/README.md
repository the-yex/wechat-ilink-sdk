# WeChat iLink SDK 示例

本目录包含微信 iLink SDK 的完整示例代码，涵盖从基础登录到高级功能的各个方面。

## 示例列表

| 示例 | 难度 | 描述 |
|------|------|------|
| [simple-login](./simple-login) | 入门 | 最简单的扫码登录 |
| [qrcode-login](./qrcode-login) | 入门 | 显式登录 + 单账号 Token 持久化 |
| [basic-bot](./basic-bot) | 进阶 | 生产配置风格的 Echo 机器人 |
| [auto-relogin](./auto-relogin) | 进阶 | 自定义会话过期回调与重登录 |
| [sqlite-storage](./sqlite-storage) | 进阶 | SQLite 存储 Token |
| [token-provider](./token-provider) | 进阶 | 外部凭据仓库 / 数据库接入模式 |
| [event-demo](./event-demo) | 进阶 | 事件系统使用示例 |
| [error-handling](./error-handling) | 进阶 | 结构化错误分类与处理策略 |
| [plugins](./plugins) | 高级 | 插件开发示例 |
| [ai-assistant](./ai-assistant) | 高级 | AI 助手集成模式 |

## 快速开始

### 1. simple-login - 最简单的登录

**适合：** 第一次使用 SDK

```bash
go run ./examples/simple-login/main.go
```

**功能：**
- 创建客户端（无 Token 存储）
- 生成并显示二维码 URL
- 等待用户扫码确认
- 输出登录结果

### 2. qrcode-login - 登录 + 单账号 Token 持久化

**适合：** 学习显式保存和恢复 Token

```bash
go run ./examples/qrcode-login/main.go
```

**功能：**
- 创建客户端和文件 Token 存储
- 生成并显示二维码
- 演示 `login.SaveDefaultToken` / `login.LoadDefaultToken`
- 演示 `client.RestoreToken`

### 3. basic-bot - 生产配置风格的 Echo 机器人

**适合：** 学习推荐的自动登录、限流、重试和自定义 HTTP client 配置

```bash
go run ./examples/basic-bot/main.go
```

**功能：**
- 自动加载存储 Token，必要时自动扫码登录
- 演示 `WithRetry`、`WithRateLimit`
- 演示 API / 长轮询 / CDN 的独立 `http.Client`
- 支持 Ctrl+C 优雅退出

### 4. auto-relogin - 自定义重登录回调

**适合：** 需要在会话过期时接入自己的通知或重登录逻辑

```bash
go run ./examples/auto-relogin/main.go
```

**功能：**
- 自定义 `WithOnSessionExpired`
- 会话过期时重新展示二维码
- 无缝恢复消息处理

### 5. sqlite-storage - SQLite 存储

**适合：** 学习自定义 TokenStore 实现

```bash
go run ./examples/sqlite-storage/main.go
```

**功能：**
- 实现 TokenStore 接口
- SQLite 数据库持久化
- 完整的消息处理示例

### 6. token-provider - 外部凭据仓库接入

**适合：** Token 保存在数据库、Redis、密钥管理系统，而不是本地文件

```bash
go run ./examples/token-provider/main.go
```

**功能：**
- 演示 `WithTokenProvider`
- 演示 `WithOnLoginSuccess`
- 演示 `WithOnTokenInvalid`
- 用内存仓库模拟数据库 / 远程凭据服务

### 7. event-demo - 事件系统

**适合：** 学习事件订阅和生命周期管理

```bash
go run ./examples/event-demo/main.go
```

**功能：**
- 订阅登录成功事件
- 订阅会话过期事件
- 订阅连接/断开事件
- 订阅错误事件

**事件用途：**
| 事件 | 用途 |
|------|------|
| `EventTypeLogin` | 记录用户信息、初始化业务状态 |
| `EventTypeSessionExpired` | 清理缓存、发送告警通知 |
| `EventTypeConnected` | 更新服务状态为"在线" |
| `EventTypeDisconnected` | 更新服务状态为"离线" |
| `EventTypeError` | 统一错误处理、监控上报 |

### 8. error-handling - 结构化错误处理

**适合：** 想知道不同错误该重试、告警还是等待用户重新触发

```bash
go run ./examples/error-handling/main.go
```

**功能：**
- 演示 `ErrContextTokenRequired`
- 演示 `IsAuthenticationError`
- 演示 `IsTemporaryError`
- 演示 `ErrorCode`

### 9. plugins - 插件开发

**适合：** 学习插件系统开发

```bash
go run ./examples/plugins/main.go
```

**功能：**
- 日志记录插件
- 命令处理插件（/help, /ping, /echo）
- 关键词自动回复插件

详细文档：[plugins/README.md](./plugins/README.md)

### 10. ai-assistant - AI 助手

**适合：** 集成 AI 服务

```bash
go run ./examples/ai-assistant/main.go
```

**功能：**
- 集成 AI API
- 上下文管理
- 流式响应

## 登录流程说明

SDK 自动处理登录流程，开箱即用：

```
┌─────────────────┐
│   创建客户端     │
│  (带 TokenStore) │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   调用 Run()    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐     有存储的 Token
│  检查 TokenStore │ ──────────────────┐
└────────┬────────┘                    │
         │ 无 Token                    │
         ▼                             ▼
┌─────────────────┐           ┌─────────────────┐
│  自动显示二维码  │           │  验证 Token      │
│  (终端打印)     │           │  (getConfig API) │
└────────┬────────┘           └────────┬────────┘
         │                             │
         ▼                             ▼
┌─────────────────┐           ┌─────────────────┐
│  用户扫码确认    │           │  Token 有效？    │
└────────┬────────┘           └────────┬────────┘
         │                      是 │        │ 否
         ▼                         ▼        ▼
┌─────────────────┐           ┌─────────┐ ┌─────────┐
│  自动保存 Token  │           │ 跳过扫码 │ │重新扫码 │
│  到 TokenStore   │           │ 直接运行 │ │  登录   │
└────────┬────────┘           └─────────┘ └─────────┘
         │
         ▼
┌─────────────────┐
│  开始消息监听    │
│  (长轮询)       │
└─────────────────┘
```

**默认行为**：
- **OnLogin**: 自动在终端显示二维码
- **OnSessionExpired**: 会话过期时自动提示重新扫码

**自定义行为**：使用 `WithOnLogin` 和 `WithOnSessionExpired` 选项覆盖默认行为。

## Token 存储

SDK 提供两种 Token 存储方式：

### MemoryTokenStore（内存存储）

```go
tokenStore := login.NewMemoryTokenStore()
```

- 仅内存存储，程序退出后丢失
- 适合测试环境

### FileTokenStore（文件存储，推荐）

```go
// 默认存储到当前目录的 .weixin 文件夹
tokenStore, err := login.NewFileTokenStore("")

// 指定自定义目录
tokenStore, err := login.NewFileTokenStore("/path/to/tokens")
```

- 持久化存储，支持自动登录
- 文件权限 0600，保护敏感信息

## 二维码显示

SDK 提供多种二维码显示方式：

### 方式 1: 终端二维码图片

```go
login.PrintQRCodeWithTerm(qr)
```

直接在终端打印 ASCII 二维码，用户可直接扫描。

### 方式 2: 打印 URL

```go
fmt.Println("打开此链接扫码：")
fmt.Println(qr.ImageURL)
```

用户在浏览器打开链接，显示二维码图片。

### 方式 3: 自定义处理

```go
client.Login(ctx, func(ctx context.Context, qr *login.QRCode) error {
    // qr.Content - 二维码内容（轮询标识）
    // qr.ImageURL - 二维码图片 URL
    // qr.StartedAt - 创建时间（用于计算过期）

    // 自定义显示逻辑
    return nil
})
```

## 消息处理

### 收消息

```go
client.Run(ctx, func(ctx context.Context, msg *ilink.Message) error {
    // 消息类型判断
    if msg.MessageType == ilink.MessageTypeUser {
        // 用户消息
    }

    // 获取文本内容
    text := msg.GetText()

    // 获取媒体内容
    media := msg.GetFirstMediaItem()

    return nil
})
```

### 发消息

```go
// 发送文本
client.SendText(ctx, msg.FromUserID, "收到消息了")

// 发送图片
client.SendImage(ctx, msg.FromUserID, imageData)

// 发送文件
client.SendFile(ctx, msg.FromUserID, fileName, fileData)
```

## 常见问题

### Q: 二维码过期怎么办？

A: 二维码有效期 5 分钟。过期后 SDK 会自动重新获取并显示新的二维码。

### Q: Token 多久过期？

A: Token 有效期由微信服务器控制。过期时 SDK 会自动提示重新扫码登录，无需手动处理。

### Q: 如何多账号登录？

A: 每次扫码登录会生成不同的 Token，使用不同的 TokenStore 实例管理：

```go
tokenStore1, _ := login.NewFileTokenStore("./account1")
tokenStore2, _ := login.NewFileTokenStore("./account2")
```

### Q: 发送消息失败怎么办？

A: 常见原因：
1. **缺少 context_token** - 确保先收到用户消息再回复
2. **Token 过期** - 触发重新登录
3. **用户 ID 错误** - 使用 `msg.FromUserID`

### Q: 如何调试？

A: 启用 DEBUG 日志：

```go
logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))
```
