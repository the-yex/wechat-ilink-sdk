package middleware

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/the-yex/wechat-ilink-sdk/ilink"
)

func TestChain(t *testing.T) {
	var order []string

	middleware1 := func(next Handler) Handler {
		return func(ctx context.Context, req *ilink.SendMessageRequest) error {
			order = append(order, "m1-before")
			err := next(ctx, req)
			order = append(order, "m1-after")
			return err
		}
	}

	middleware2 := func(next Handler) Handler {
		return func(ctx context.Context, req *ilink.SendMessageRequest) error {
			order = append(order, "m2-before")
			err := next(ctx, req)
			order = append(order, "m2-after")
			return err
		}
	}

	final := func(ctx context.Context, req *ilink.SendMessageRequest) error {
		order = append(order, "final")
		return nil
	}

	handler := Chain(final, middleware1, middleware2)
	err := handler(context.Background(), &ilink.SendMessageRequest{})

	require.NoError(t, err)
	assert.Equal(t, []string{
		"m1-before",
		"m2-before",
		"final",
		"m2-after",
		"m1-after",
	}, order)
}

func TestRetry(t *testing.T) {
	t.Run("success on first try", func(t *testing.T) {
		calls := 0
		handler := func(ctx context.Context, req *ilink.SendMessageRequest) error {
			calls++
			return nil
		}

		retryMiddleware := Retry(RetryConfig{
			MaxAttempts: 3,
			WaitMin:     1 * time.Millisecond,
			WaitMax:     5 * time.Millisecond,
			Retryable:   func(err error) bool { return true },
		})

		wrapped := retryMiddleware(handler)
		err := wrapped(context.Background(), &ilink.SendMessageRequest{})

		require.NoError(t, err)
		assert.Equal(t, 1, calls)
	})

	t.Run("success on retry", func(t *testing.T) {
		calls := 0
		handler := func(ctx context.Context, req *ilink.SendMessageRequest) error {
			calls++
			if calls < 3 {
				return errors.New("temporary error")
			}
			return nil
		}

		retryMiddleware := Retry(RetryConfig{
			MaxAttempts: 3,
			WaitMin:     1 * time.Millisecond,
			WaitMax:     5 * time.Millisecond,
			Retryable:   func(err error) bool { return true },
		})

		wrapped := retryMiddleware(handler)
		err := wrapped(context.Background(), &ilink.SendMessageRequest{})

		require.NoError(t, err)
		assert.Equal(t, 3, calls)
	})

	t.Run("max attempts exceeded", func(t *testing.T) {
		calls := 0
		expectedErr := errors.New("persistent error")
		handler := func(ctx context.Context, req *ilink.SendMessageRequest) error {
			calls++
			return expectedErr
		}

		retryMiddleware := Retry(RetryConfig{
			MaxAttempts: 3,
			WaitMin:     1 * time.Millisecond,
			WaitMax:     5 * time.Millisecond,
			Retryable:   func(err error) bool { return true },
		})

		wrapped := retryMiddleware(handler)
		err := wrapped(context.Background(), &ilink.SendMessageRequest{})

		require.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Equal(t, 3, calls)
	})

	t.Run("non-retryable error", func(t *testing.T) {
		calls := 0
		expectedErr := errors.New("non-retryable")
		handler := func(ctx context.Context, req *ilink.SendMessageRequest) error {
			calls++
			return expectedErr
		}

		retryMiddleware := Retry(RetryConfig{
			MaxAttempts: 3,
			WaitMin:     1 * time.Millisecond,
			WaitMax:     5 * time.Millisecond,
			Retryable:   func(err error) bool { return false },
		})

		wrapped := retryMiddleware(handler)
		err := wrapped(context.Background(), &ilink.SendMessageRequest{})

		require.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Equal(t, 1, calls)
	})

	t.Run("default config skips context cancellation", func(t *testing.T) {
		calls := 0
		handler := func(ctx context.Context, req *ilink.SendMessageRequest) error {
			calls++
			return context.Canceled
		}

		wrapped := Retry(DefaultRetryConfig())(handler)
		err := wrapped(context.Background(), &ilink.SendMessageRequest{})

		require.ErrorIs(t, err, context.Canceled)
		assert.Equal(t, 1, calls)
	})
}

func TestRecovery(t *testing.T) {
	// Create a real logger that outputs to stdout
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	t.Run("no panic", func(t *testing.T) {
		handler := func(ctx context.Context, req *ilink.SendMessageRequest) error {
			return nil
		}

		recoveryMiddleware := Recovery(logger)
		wrapped := recoveryMiddleware(handler)
		err := wrapped(context.Background(), &ilink.SendMessageRequest{})

		require.NoError(t, err)
	})

	t.Run("panic recovered", func(t *testing.T) {
		handler := func(ctx context.Context, req *ilink.SendMessageRequest) error {
			panic("test panic")
		}

		recoveryMiddleware := Recovery(logger)
		wrapped := recoveryMiddleware(handler)
		err := wrapped(context.Background(), &ilink.SendMessageRequest{})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "panic")
	})

	t.Run("non-string panic recovered", func(t *testing.T) {
		handler := func(ctx context.Context, req *ilink.SendMessageRequest) error {
			panic(123)
		}

		recoveryMiddleware := Recovery(logger)
		wrapped := recoveryMiddleware(handler)
		err := wrapped(context.Background(), &ilink.SendMessageRequest{})

		require.Error(t, err)
		assert.Equal(t, "panic: 123", err.Error())
	})
}

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	assert.Equal(t, 3, config.MaxAttempts)
	assert.Equal(t, 1*time.Second, config.WaitMin)
	assert.Equal(t, 5*time.Second, config.WaitMax)
	assert.NotNil(t, config.Retryable)
}

func TestDefaultRateLimitConfig(t *testing.T) {
	config := DefaultRateLimitConfig()

	assert.Equal(t, 5, config.MessagesPerSecond)
	assert.Equal(t, 1, config.Burst)
}

func TestRateLimit(t *testing.T) {
	t.Run("throttles sequential calls", func(t *testing.T) {
		handler := func(ctx context.Context, req *ilink.SendMessageRequest) error {
			return nil
		}

		wrapped := RateLimit(RateLimitConfig{
			MessagesPerSecond: 50,
			Burst:             1,
		})(handler)

		start := time.Now()
		require.NoError(t, wrapped(context.Background(), &ilink.SendMessageRequest{}))
		require.NoError(t, wrapped(context.Background(), &ilink.SendMessageRequest{}))

		assert.GreaterOrEqual(t, time.Since(start), 18*time.Millisecond)
	})

	t.Run("respects canceled context", func(t *testing.T) {
		handler := func(ctx context.Context, req *ilink.SendMessageRequest) error {
			return nil
		}

		wrapped := RateLimit(RateLimitConfig{
			MessagesPerSecond: 1,
			Burst:             1,
		})(handler)

		require.NoError(t, wrapped(context.Background(), &ilink.SendMessageRequest{}))

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := wrapped(ctx, &ilink.SendMessageRequest{})
		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
	})
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"context canceled", context.Canceled, false},
		{"context deadline", context.DeadlineExceeded, false},
		{"other error", errors.New("some error"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsRetryableError(tt.err))
		})
	}
}
