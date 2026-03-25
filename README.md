# WeChat iLink SDK for Go

[English](./README.md) | [中文文档](./README_zh_CN.md)

[![Go Reference](https://pkg.go.dev/badge/github.com/the-yex/wechat-ilink-sdk.svg)](https://pkg.go.dev/github.com/the-yex/wechat-ilink-sdk)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A professional, highly extensible Go SDK for building WeChat bot applications based on the iLink protocol.

## Features

- **QR Code Login** - Scan QR code to authenticate, tokens persisted locally
- **Auto Re-login** - Automatically validates stored tokens and handles session expiry
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
)

func main() {
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

    // Create token store for persistence (auto-login support)
    tokenStore, _ := login.NewFileTokenStore("")

    // Create client
    client, _ := ilinksdk.NewClient(
        ilinksdk.WithLogger(logger),
        ilinksdk.WithTokenStore(tokenStore),
    )
    defer client.Close()

    // Login via QR code (SDK validates stored token first)
    result, err := client.Login(context.Background(), func(ctx context.Context, qr *login.QRCode) error {
        // Display QR code for user to scan
        login.PrintQRCodeWithTerm(qr)
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
        // Auto-reply to text messages
        if text := msg.GetText(); text != "" {
            return client.SendText(ctx, msg.FromUserID, "Echo: "+text)
        }
        return nil
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
├── login/             # Login service & token storage
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
    ilinksdk.WithTokenStore(tokenStore),
    ilinksdk.WithPlugins(myPlugin1, myPlugin2),
)
```

## Token Management

### Auto-login

The SDK automatically handles token persistence:

1. **First Run**: Display QR code → User scans → Token saved to store
2. **Subsequent Runs**: SDK loads stored token → Validates with API → Skip QR code if valid

```go
// FileTokenStore saves tokens to ~/.wechat-ilink/tokens.json by default
tokenStore, _ := login.NewFileTokenStore("")

// Or specify custom directory
tokenStore, _ := login.NewFileTokenStore("/path/to/tokens")

// Or use in-memory store (no persistence)
tokenStore := login.NewMemoryTokenStore()
```

### Token Validation

The SDK validates stored tokens on startup using the `getConfig` API. If the token is expired or invalid, it automatically clears the stored token and prompts for QR code scan.

### Session Expiry Handling

```go
// Set callback for session expiry
client.SetOnSessionExpired(func(ctx context.Context) (*ilink.LoginResult, error) {
    fmt.Println("Session expired! Please re-scan QR code")
    return client.Login(ctx, func(ctx context.Context, qr *login.QRCode) error {
        login.PrintQRCodeWithTerm(qr)
        return nil
    })
})
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
client.OnMessage(func(ctx context.Context, e *event.Event) error {
    msg := e.Data.(*ilink.Message)
    log.Printf("Received: %s", msg.GetText())
    return nil
})

client.OnSessionExpired(func(ctx context.Context, e *event.Event) error {
    log.Println("Session expired, need to re-login")
    return nil
})
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

// Register plugin
client.UsePlugin(context.Background(), &MyPlugin{})
```

See [examples/plugins/README.md](./examples/plugins/README.md) for detailed plugin development guide.

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

| Example | Description |
|---------|-------------|
| `simple-login` | Basic QR code login |
| `qrcode-login` | Login with token storage |
| `qrcode-login-with-image` | Full bot with auto-reply |
| `auto-relogin` | Auto re-login on session expiry |
| `basic-bot` | Echo bot with middleware |
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