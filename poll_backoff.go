package ilinksdk

import (
	"context"
	"math/rand"
	"time"
)

func (c *Client) waitPollErrorBackoff(ctx context.Context, consecutiveErrors int, err error) error {
	wait := calculatePollErrorBackoff(c.config.PollErrorBackoffMin, c.config.PollErrorBackoffMax, consecutiveErrors)

	c.config.Logger.Warn("backing off after long-poll failure",
		"consecutive_errors", consecutiveErrors,
		"backoff", wait,
		"temporary", IsTemporaryError(err),
	)

	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func calculatePollErrorBackoff(min, max time.Duration, consecutiveErrors int) time.Duration {
	if consecutiveErrors < 1 {
		consecutiveErrors = 1
	}

	wait := min * time.Duration(1<<(consecutiveErrors-1))
	if wait > max {
		wait = max
	}

	if wait <= 0 {
		return 0
	}

	jitter := wait / 10
	if jitter <= 0 {
		return wait
	}

	delta := time.Duration(rand.Int63n(int64(jitter*2)+1)) - jitter
	wait += delta
	if wait < 0 {
		return 0
	}
	return wait
}
