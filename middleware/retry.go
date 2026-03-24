package middleware

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"github.com/the-yex/wechat-ilink-sdk/ilink"
)

// RetryConfig holds retry configuration.
type RetryConfig struct {
	MaxAttempts int           // Maximum number of attempts (including initial)
	WaitMin     time.Duration // Minimum wait time between retries
	WaitMax     time.Duration // Maximum wait time between retries
	Retryable   func(error) bool // Function to determine if error is retryable
}

// DefaultRetryConfig returns a default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts: 3,
		WaitMin:     1 * time.Second,
		WaitMax:     5 * time.Second,
		Retryable:   DefaultRetryable,
	}
}

// DefaultRetryable determines if an error is retryable.
func DefaultRetryable(err error) bool {
	// Retry on network errors, 5xx errors, etc.
	// Don't retry on 4xx errors or session errors
	if apiErr, ok := err.(*ilink.APIError); ok {
		// Don't retry client errors
		if apiErr.Code >= 400 && apiErr.Code < 500 {
			return false
		}
		// Retry server errors
		return apiErr.Code >= 500
	}
	// Retry unknown errors (network issues, etc.)
	return true
}

// IsRetryableError determines if an error is retryable.
// It returns false for context errors (Canceled, DeadlineExceeded) and true for most other errors.
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}
	// Don't retry context errors
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	return DefaultRetryable(err)
}

// Retry returns a retry middleware.
func Retry(cfg RetryConfig) Middleware {
	if cfg.MaxAttempts < 1 {
		cfg.MaxAttempts = 1
	}
	if cfg.WaitMin == 0 {
		cfg.WaitMin = 1 * time.Second
	}
	if cfg.WaitMax == 0 {
		cfg.WaitMax = 5 * time.Second
	}
	if cfg.Retryable == nil {
		cfg.Retryable = DefaultRetryable
	}

	return func(next Handler) Handler {
		return func(ctx context.Context, req *ilink.SendMessageRequest) error {
			var lastErr error

			for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
				err := next(ctx, req)
				if err == nil {
					return nil
				}

				// Check if error is retryable
				if !cfg.Retryable(err) {
					return err
				}

				lastErr = err

				// Wait before next attempt (except for last attempt)
				if attempt < cfg.MaxAttempts {
					wait := calculateBackoff(cfg.WaitMin, cfg.WaitMax, attempt)
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(wait):
					}
				}
			}

			return lastErr
		}
	}
}

// calculateBackoff calculates exponential backoff with jitter.
func calculateBackoff(min, max time.Duration, attempt int) time.Duration {
	// Exponential backoff
	wait := min * time.Duration(1<<(attempt-1))
	if wait > max {
		wait = max
	}

	// Add jitter (±10%)
	jitter := time.Duration(rand.Float64() * 0.2 * float64(wait))
	wait = wait - wait/10 + jitter

	return wait
}