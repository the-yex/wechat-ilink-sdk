package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"

	ilinksdk "github.com/the-yex/wechat-ilink-sdk"
)

func main() {
	ctx := context.Background()

	fmt.Println("=== WeChat iLink SDK - Error Handling Example ===")
	fmt.Println()

	printClassification("missing context token", missingContextTokenError(ctx))
	printClassification("authentication failure", sendWithMockStatus(ctx, http.StatusUnauthorized))
	printClassification("temporary server failure", sendWithMockStatus(ctx, http.StatusServiceUnavailable))
}

func missingContextTokenError(ctx context.Context) error {
	client, err := ilinksdk.NewClient()
	if err != nil {
		return err
	}
	defer client.Close()

	return client.SendText(ctx, "user-1", "hello")
}

func sendWithMockStatus(ctx context.Context, statusCode int) error {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ilink/bot/sendmessage" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(`{"ret":1}`))
	}))
	defer server.Close()

	client, err := ilinksdk.NewClient(
		ilinksdk.WithBaseURL(server.URL),
		ilinksdk.WithToken("test-token"),
	)
	if err != nil {
		return err
	}
	defer client.Close()

	client.SetContextToken("user-1", "ctx-token")
	return client.SendText(ctx, "user-1", "hello")
}

func printClassification(name string, err error) {
	fmt.Printf("Case: %s\n", name)
	if err == nil {
		fmt.Println("  result: success")
		fmt.Println()
		return
	}

	fmt.Printf("  error: %v\n", err)

	switch {
	case errors.Is(err, ilinksdk.ErrContextTokenRequired):
		fmt.Println("  action: wait for the user to message first, then reply with the stored context token")
	case ilinksdk.IsAuthenticationError(err):
		fmt.Println("  action: stop retrying and trigger login / token refresh")
	case ilinksdk.IsTemporaryError(err):
		fmt.Println("  action: retry with backoff or hand over to background retry logic")
	default:
		fmt.Println("  action: inspect and decide based on business policy")
	}

	if code, ok := ilinksdk.ErrorCode(err); ok {
		fmt.Printf("  normalized code: %d\n", code)
	}

	fmt.Println()
}
