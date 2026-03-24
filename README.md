# WeChat iLink SDK for Go

[![Go Reference](https://pkg.go.dev/badge/github.com/the-yex/wechat-ilink-sdk.svg)](https://pkg.go.dev/github.com/the-yex/wechat-ilink-sdk)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A professional, highly extensible Go SDK for building WeChat bot applications based on the iLink protocol.

## Features

- **QR Code Login** - Scan QR code to authenticate, tokens persisted locally
- **Message Handling** - Receive and send text, image, video, file, and voice messages
- **CDN Media** - AES-128-ECB encrypted upload/download for media files
- **Middleware System** - Built-in logging, retry, and recovery middleware
- **Plugin System** - Extensible plugin architecture for custom functionality
- **Event System** - Asynchronous event dispatching for loose coupling
- **Session Management** - Automatic handling of context tokens and session expiry
- **Production Ready** - Comprehensive error handling, logging, and testing

## Architecture

```
Client (Facade)
├── MessageService   → SendText, SendImage, SendTyping
├── MediaService     → Upload, Download
├── AuthService      → Login, SetToken, LoadToken
├── SessionService   → IsPaused, RemainingPause
└── EventDispatcher  → Subscribe, Dispatch
```

The SDK follows a layered architecture with clear separation of concerns:
- **Client** - Unified entry point (Facade pattern)
- **Services** - Domain-specific business logic (Message, Media, Auth, Session)
- **iLink** - Protocol layer (packet handling, connection management)
- **Transport** - Network abstraction

## Installation

```bash
go get github.com/the-yex/wechat-ilink-sdk
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log/slog"
    "os"

    "github.com/the-yex/wechat-ilink-sdk"
    "github.com/the-yex/wechat-ilink-sdk/ilink"
    "github.com/the-yex/wechat-ilink-sdk/login"
    "github.com/the-yex/wechat-ilink-sdk/middleware"
)

func main() {
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

    // Create token store for persistence
    tokenStore, _ := login.NewFileTokenStore("")

    // Create client
    client, _ := ilinksdk.NewClient(
        ilinksdk.WithLogger(logger),
        ilinksdk.WithTokenStore(tokenStore),
        ilinksdk.WithMiddleware(middleware.Logging(logger)),
    )
    defer client.Close()

    // Login via QR code (token is saved automatically)
    result, err := client.Login(context.Background(), func(ctx context.Context, qr *login.QRCode) error {
        fmt.Println("Scan this QR code:")
        fmt.Println(qr.Content)
        return nil
    })
    if err != nil {
        logger.Error("login failed", "error", err)
        os.Exit(1)
    }
    logger.Info("logged in", "account", result.AccountID)

    // Run bot
    ctx := context.Background()
    client.Run(ctx, func(ctx context.Context, msg *ilink.Message) error {
        return client.SendText(ctx, msg.FromUserID, "Hello!")
    })
}
```

## Project Structure

```
wechat-ilink-sdk/
├── client.go          # Main client (entry point)
├── config.go          # Configuration
├── options.go         # Option pattern
├── errors.go          # Error definitions
├── version.go         # Version info
├── service/           # Service layer (Message, Media, Auth, Session)
├── types/             # Request/Response types
├── ilink/             # iLink protocol layer
├── login/             # Login service
├── media/             # CDN media handling
├── middleware/        # Middleware system
├── plugin/            # Plugin system
├── event/             # Event system
├── internal/          # Internal implementation
└── examples/          # Example code
```

## Configuration

```go
client, err := ilinksdk.NewClient(
    ilinksdk.WithBaseURL("https://ilinkai.weixin.qq.com"),
    ilinksdk.WithCDNBaseURL("https://novac2c.cdn.weixin.qq.com/c2c"),
    ilinksdk.WithTimeout(30 * time.Second),
    ilinksdk.WithRetry(3, time.Second, 5 * time.Second),
    ilinksdk.WithLogger(slog.Default()),
)
```

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

Subscribe to SDK events for reactive programming:

### Available Events

| Event | When Fired | Data Type |
|-------|------------|-----------|
| `EventTypeMessage` | New message received | `*ilink.Message` |
| `EventTypeLogin` | Login successful | `*ilink.LoginResult` |
| `EventTypeError` | Error occurred | `error` |
| `EventTypeSessionExpired` | Session expired | `nil` |
| `EventTypeConnected` | Client started | `nil` |
| `EventTypeDisconnected` | Client stopped | `nil` |

### Using Event Handlers

```go
// Subscribe using convenience methods
client.OnMessage(func(ctx context.Context, e *event.Event) error {
    msg := e.Data.(*ilink.Message)
    log.Printf("收到消息: %s", msg.Content)
    return nil
})

client.OnSessionExpired(func(ctx context.Context, e *event.Event) error {
    log.Println("Session 过期，需要重新登录")
    // Auto re-login logic
    return nil
})

client.OnError(func(ctx context.Context, e *event.Event) error {
    err := e.Data.(error)
    log.Printf("错误: %v", err)
    return nil
})
```

### Advanced Usage

```go
// Direct dispatcher access for more control
client.Events().Subscribe(event.EventTypeMessage, handler)
client.Events().Unsubscribe(event.EventTypeMessage)

// Synchronous dispatch (blocks until all handlers complete)
client.Events().DispatchSync(ctx, &event.Event{...})
```

## Plugin System

Create custom plugins to extend SDK functionality:

```go
type MyPlugin struct{}

func (p *MyPlugin) Name() string { return "my-plugin" }
func (p *MyPlugin) Initialize(ctx context.Context, sdk plugin.SDK) error { return nil }
func (p *MyPlugin) OnMessage(ctx context.Context, msg *ilink.Message) error {
    // Process message
    return nil
}
func (p *MyPlugin) OnError(ctx context.Context, err error) {}

client.UsePlugin(&MyPlugin{})
```

## CDN Media

### Upload Media

```go
result, err := client.UploadMedia(ctx, &media.UploadRequest{
    Data:      imageData,
    MediaType: ilink.UploadMediaTypeImage,
    ToUserID:  "user-id",
})
```

### Download Media

```go
data, err := client.DownloadMedia(ctx, &media.DownloadRequest{
    EncryptQueryParam: "cdn-param",
    AESKey:            "base64-encoded-key",
})
```

## Error Handling

```go
import "errors"

err := client.SendText(ctx, toUserID, text)
if errors.Is(err, ilinksdk.ErrSessionExpired) {
    // Session expired, need to re-login
}
if errors.Is(err, ilinksdk.ErrContextTokenRequired) {
    // No context token for this user
}
```

## Examples

See the [examples](./examples/) directory:

- `basic-bot` - Echo bot with QR code login
- `ai-assistant` - AI assistant integration pattern

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