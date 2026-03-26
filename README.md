# WeChat iLink SDK for Go

[English](./README.md) | [中文文档](./README_zh_CN.md)

[![Go Reference](https://pkg.go.dev/badge/github.com/the-yex/wechat-ilink-sdk.svg)](https://pkg.go.dev/github.com/the-yex/wechat-ilink-sdk)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A professional, highly extensible Go SDK for building WeChat bot applications based on the iLink protocol.

## Features

- **QR Code Login** - Scan QR code to authenticate, tokens persisted locally
- **Auto Re-login** - Automatically validates stored tokens and handles session expiry
- **Message Handling** - Receive and send text, image, video, file, and voice messages
- **Middleware System** - Built-in logging, retry, and recovery middleware
- **Plugin System** - Extensible plugin architecture for custom functionality
- **Event System** - Asynchronous event dispatching for loose coupling
- **Production Ready** - Comprehensive error handling, logging, and testing

## Installation

```bash
go get github.com/the-yex/wechat-ilink-sdk
```

## Quick Start

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

    // Create token store
    tokenStore, _ := login.NewFileTokenStore("")

    // Create client
    client, _ := ilinksdk.NewClient(
        ilinksdk.WithLogger(logger),
        ilinksdk.WithTokenStore(tokenStore),
    )
    defer client.Close()

    // Run the bot - SDK handles login and messages automatically
    ctx := context.Background()
    client.Run(ctx, func(ctx context.Context, msg *ilink.Message) error {
        if text := msg.GetText(); text != "" {
            return client.SendText(ctx, msg.FromUserID, "Echo: "+text)
        }
        return nil
    })
}
```

## Interaction Flow

### Login Flow

```
┌─────────────┐     Get QR Code      ┌─────────────┐
│    User     │ ◄────────────────── │ iLink Server │
│             │                     │             │
│  Scan &     │ ──────────────────► │             │
│  Confirm    │                     │             │
└─────────────┘                     └─────────────┘
       │
       ▼
  Get bot_token
  Auto-save to TokenStore
```

SDK handles: Display QR code → Poll scan status → Save credentials.

### Message Flow

```
┌──────────┐                    ┌──────────┐
│ WeChat   │ ──Send message────► │ iLink    │
│ User     │                    │ Server   │
└──────────┘                    └────┬─────┘
                                     │
                                     ▼
┌──────────┐                    ┌──────────┐
│ Bot SDK  │ ◄─Long-poll msgs─── │ iLink    │
│          │ ───Send reply──────► │ Server   │
└──────────┘                    └──────────┘
       │
       ▼
  User receives reply
```

**Key Concepts**:
- SDK receives messages via long-polling, not WebSocket
- `context_token` is managed automatically for each message

### Message Types

| type | Type | Description |
|------|------|-------------|
| `1` | TEXT | Text message |
| `2` | IMAGE | Image message |
| `3` | VOICE | Voice message |
| `4` | FILE | File message |
| `5` | VIDEO | Video message |

## Sending Messages

```go
// Send text
client.SendText(ctx, toUserID, "Hello!")

// Send image
client.SendImage(ctx, toUserID, imageData)

// Send video
client.SendVideo(ctx, toUserID, videoData)

// Send file
client.SendFile(ctx, toUserID, "document.pdf", fileData)

// Send typing indicator
client.SendTyping(ctx, toUserID, true)  // Start typing
client.SendTyping(ctx, toUserID, false) // Stop typing
```

## Receiving Messages

### Option 1: Register Type-Specific Handlers (Recommended)

Cleaner code with separate handlers for each message type:

```go
client.OnText(func(ctx context.Context, msg *ilink.Message, text string) error {
    fmt.Printf("Received text: %s\n", text)
    return client.SendText(ctx, msg.FromUserID, "Echo: "+text)
})

client.OnImage(func(ctx context.Context, msg *ilink.Message, item *types.ImageItem) error {
    fmt.Printf("Received image\n")
    // Download and reply with image
    data, _ := client.DownloadMedia(ctx, &media.DownloadRequest{
        EncryptQueryParam: item.Media.EncryptQueryParam,
        AESKey:            item.Media.AESKey,
    })
    return client.SendImage(ctx, msg.FromUserID, data)
})

client.OnVideo(func(ctx context.Context, msg *ilink.Message, item *types.VideoItem) error {
    fmt.Printf("Received video\n")
    return nil
})

client.OnVoice(func(ctx context.Context, msg *ilink.Message, item *types.VoiceItem) error {
    fmt.Printf("Received voice: %s\n", item.Text)
    return nil
})

client.OnFile(func(ctx context.Context, msg *ilink.Message, item *types.FileItem) error {
    fmt.Printf("Received file: %s\n", item.FileName)
    return nil
})

// Run the bot (no handler needed, uses registered handlers)
client.Run(ctx, nil)
```

### Option 2: General Message Handler

For scenarios where you want to handle all message types in one place:

```go
client.OnMessage(func(ctx context.Context, msg *ilink.Message) error {
    if !msg.IsFromUser() {
        return nil
    }

    // Check msg.ItemList to determine message type
    if text := msg.GetText(); text != "" {
        return client.SendText(ctx, msg.FromUserID, "Echo: "+text)
    }

    return nil
})

client.Run(ctx, nil)
```

### Option 3: Pass to Run() Directly

Simplest approach:

```go
client.Run(ctx, func(ctx context.Context, msg *ilink.Message) error {
    if text := msg.GetText(); text != "" {
        return client.SendText(ctx, msg.FromUserID, "Echo: "+text)
    }
    return nil
})
```

## Configuration

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

## Token Management

SDK uses file storage by default, automatically handling token persistence:

```go
// Default: stored in ./.weixin/default.json
tokenStore, _ := login.NewFileTokenStore("")

// Or specify custom directory
tokenStore, _ := login.NewFileTokenStore("./my-bot")
```

### Custom Storage

Implement the `TokenStore` interface for custom storage:

```go
type TokenStore interface {
    Get(ctx context.Context) (*TokenInfo, error)
    Set(ctx context.Context, info *TokenInfo) error
    Clear(ctx context.Context) error
}
```

See [examples/sqlite-storage](./examples/sqlite-storage/) for SQLite storage example.

## Middleware

### Built-in Middleware

| Middleware | Description |
|------------|-------------|
| `Logging` | Request/response logging |
| `Retry` | Automatic retry with exponential backoff |
| `Recovery` | Panic recovery |

### Custom Middleware

```go
func CustomMiddleware() middleware.Middleware {
    return func(next middleware.Handler) middleware.Handler {
        return func(ctx context.Context, req *ilink.SendMessageRequest) error {
            // Pre-processing
            err := next(ctx, req)
            // Post-processing
            return err
        }
    }
}

client.Use(CustomMiddleware())
```

## Event System

SDK has a built-in event system for monitoring lifecycle events.

### Available Events

| Event | When Fired |
|-------|------------|
| `EventTypeMessage` | New message received |
| `EventTypeLogin` | Login successful |
| `EventTypeError` | Error occurred |
| `EventTypeSessionExpired` | Session expired |
| `EventTypeConnected` | Client started |
| `EventTypeDisconnected` | Client stopped |

### Using Events

```go
// Subscribe to login event
client.Events().Subscribe(event.EventTypeLogin, func(ctx context.Context, e *event.Event) error {
    result := e.Data.(*ilink.LoginResult)
    log.Printf("Logged in: %s", result.UserID)
    return nil
})

// Subscribe to session expired event
client.Events().Subscribe(event.EventTypeSessionExpired, func(ctx context.Context, e *event.Event) error {
    log.Println("Session expired")
    return nil
})
```

## Plugin System

```go
type MyPlugin struct{}

func (p *MyPlugin) Name() string { return "my-plugin" }
func (p *MyPlugin) Initialize(ctx context.Context, sdk plugin.SDK) error { return nil }
func (p *MyPlugin) OnMessage(ctx context.Context, msg *ilink.Message) error {
    // Process message
    return nil
}
func (p *MyPlugin) OnError(ctx context.Context, err error) {}
```

### Registering Plugins

Both methods work the same way, executing plugins in registration order:

**Option 1: Batch Registration (Recommended)**

```go
client, _ := ilinksdk.NewClient(
    ilinksdk.WithPlugins(plugin1, plugin2, plugin3),
)
```

**Option 2: Individual Registration**

```go
client.UsePlugin(context.Background(), plugin1)
client.UsePlugin(context.Background(), plugin2)
```

### Built-in Plugins

SDK provides the following built-in plugins:

#### LogoutPlugin - Logout Command Plugin

Users can logout by sending `/exit` command. SDK clears the stored token and automatically shows a QR code for re-login:

```go
// Simple usage - direct registration
client, _ := ilinksdk.NewClient(
    ilinksdk.WithPlugins(plugin.NewLogoutPlugin()),
)

// Optional: with callback
client, _ := ilinksdk.NewClient(
    ilinksdk.WithPlugins(plugin.NewLogoutPlugin(func(ctx context.Context) error {
        log.Println("User logged out, waiting for re-scan")
        return nil
    })),
)
```

User interaction flow:
```
User: /exit
Bot:  Are you sure you want to exit? Send /exit again to confirm.
User: /exit
Bot:  Exiting, please scan QR code to login again...
[SDK automatically shows QR code for re-login]
```

See [examples/plugins/README.md](./examples/plugins/README.md) for detailed plugin development guide.

## Examples

See the [examples](./examples/) directory:

| Example | Description |
|---------|-------------|
| `simple-login` | Basic QR code login |
| `qrcode-login` | Login with token storage |
| `qrcode-login-with-image` | Full bot with auto-reply |
| `auto-relogin` | Auto re-login on session expiry |
| `sqlite-storage` | SQLite storage for user info |
| `basic-bot` | Echo bot with middleware |
| `event-demo` | Event system usage example |
| `plugins` | Plugin development examples |
| `ai-assistant` | AI assistant integration pattern |

## Development

```bash
# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Lint
golangci-lint run
```

## License

MIT License - see [LICENSE](LICENSE) for details.