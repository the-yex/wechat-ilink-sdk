package ilinksdk

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient_RejectsInvalidRetryConfig(t *testing.T) {
	_, err := NewClient(WithRetry(0, time.Second, 2*time.Second))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidConfig)
}

func TestNewClient_RejectsInvalidRateLimitConfig(t *testing.T) {
	_, err := NewClient(WithRateLimit(0, 1))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidConfig)
}

func TestNewClient_AllowsValidHighLevelMiddlewareOptions(t *testing.T) {
	client, err := NewClient(
		WithRetry(3, time.Millisecond, 2*time.Millisecond),
		WithRateLimit(5, 1),
	)
	require.NoError(t, err)

	assert.Len(t, client.middleware, 2)
}
