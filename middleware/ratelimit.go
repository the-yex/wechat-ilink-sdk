package middleware

import (
	"context"
	"sync"
	"time"

	"github.com/the-yex/wechat-ilink-sdk/ilink"
)

// RateLimitConfig controls outbound send throughput.
type RateLimitConfig struct {
	MessagesPerSecond int // Maximum sends per second.
	Burst             int // Number of sends allowed immediately.
}

// DefaultRateLimitConfig returns a conservative default outbound send limit.
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		MessagesPerSecond: 5,
		Burst:             1,
	}
}

// RateLimit returns a middleware that enforces an outbound send rate limit.
func RateLimit(cfg RateLimitConfig) Middleware {
	if cfg.MessagesPerSecond < 1 {
		cfg.MessagesPerSecond = DefaultRateLimitConfig().MessagesPerSecond
	}
	if cfg.Burst < 1 {
		cfg.Burst = DefaultRateLimitConfig().Burst
	}

	limiter := newRateLimiter(cfg.MessagesPerSecond, cfg.Burst)

	return func(next Handler) Handler {
		return func(ctx context.Context, req *ilink.SendMessageRequest) error {
			if err := limiter.Wait(ctx); err != nil {
				return err
			}
			return next(ctx, req)
		}
	}
}

type rateLimiter struct {
	mu     sync.Mutex
	rate   float64
	burst  float64
	tokens float64
	last   time.Time
}

func newRateLimiter(messagesPerSecond, burst int) *rateLimiter {
	now := time.Now()
	return &rateLimiter{
		rate:   float64(messagesPerSecond),
		burst:  float64(burst),
		tokens: float64(burst),
		last:   now,
	}
}

// Wait reserves capacity for a single send. If no token is available immediately,
// it reserves the next slot and waits exactly until that reserved slot is ready.
func (l *rateLimiter) Wait(ctx context.Context) error {
	wait := l.reserve()
	if wait <= 0 {
		return nil
	}

	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (l *rateLimiter) reserve() time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	if now.After(l.last) {
		elapsed := now.Sub(l.last).Seconds()
		l.tokens += elapsed * l.rate
		if l.tokens > l.burst {
			l.tokens = l.burst
		}
		l.last = now
	}

	if l.tokens >= 1 {
		l.tokens--
		return 0
	}

	missing := 1 - l.tokens
	wait := time.Duration((missing / l.rate) * float64(time.Second))

	// Reserve the next available slot for this caller by moving the internal
	// clock forward. Future callers will queue behind this reservation.
	l.tokens = 0
	l.last = l.last.Add(wait)

	return l.last.Sub(now)
}
