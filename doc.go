// Package ilinksdk provides a Go SDK for WeChat iLink protocol.
//
// This SDK enables developers to build WeChat bot applications with:
//   - Message receiving and sending (text, image, video, file, voice)
//   - CDN media upload/download with AES-128-ECB encryption
//   - Middleware support for logging, retry, rate limiting
//   - Plugin system for extensibility
//   - Event system for loose coupling
//
// Example usage:
//
//	client, _ := ilinksdk.NewClient(ilinksdk.WithToken("your-token"))
//	client.Run(ctx, func(ctx context.Context, msg *ilink.Message) error {
//	    return client.SendText(ctx, msg.FromUserID, "Hello!")
//	})
package ilinksdk