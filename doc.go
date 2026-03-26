// Package ilinksdk provides a Go SDK for WeChat iLink protocol.
//
// This SDK enables developers to build WeChat bot applications with:
//   - Message receiving and sending (text, image, video, file, voice)
//   - Auto-login, token persistence, session-expiry recovery, and long-poll backoff
//   - Injected HTTP clients for API, long-poll, and CDN traffic
//   - CDN media upload/download with AES-128-ECB encryption
//   - Middleware support for logging, retry, rate limiting, and recovery
//   - Graceful shutdown that drains in-flight async event handlers
//   - Plugin system and event system for extensibility
//
// Example usage:
//
//	client, _ := ilinksdk.NewClient(ilinksdk.WithToken("your-token"))
//	client.Run(ctx, func(ctx context.Context, msg *ilink.Message) error {
//	    return client.SendText(ctx, msg.FromUserID, "Hello!")
//	})
package ilinksdk
