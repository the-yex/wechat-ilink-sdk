package ilinksdk

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/the-yex/wechat-ilink-sdk/event"
	"github.com/the-yex/wechat-ilink-sdk/ilink"
	"github.com/the-yex/wechat-ilink-sdk/internal/service"
	"github.com/the-yex/wechat-ilink-sdk/login"
	"github.com/the-yex/wechat-ilink-sdk/media"
)

type testAPIClient struct {
	sendCalls int
	failUntil int
}

func (c *testAPIClient) SendMessage(ctx context.Context, req *ilink.SendMessageRequest) error {
	c.sendCalls++
	if c.sendCalls <= c.failUntil {
		return errors.New("temporary send failure")
	}
	return nil
}

func (c *testAPIClient) SendTyping(ctx context.Context, req *ilink.SendTypingRequest) error {
	return nil
}

func (c *testAPIClient) GetConfig(ctx context.Context, req *ilink.GetConfigRequest) (*ilink.GetConfigResponse, error) {
	return &ilink.GetConfigResponse{TypingTicket: "typing-ticket"}, nil
}

type noopCDNClient struct{}

func (c *noopCDNClient) Upload(ctx context.Context, req *media.UploadRequest) (*media.UploadResult, error) {
	return nil, nil
}

func (c *noopCDNClient) Download(ctx context.Context, req *media.DownloadRequest) ([]byte, error) {
	return nil, nil
}

func TestNewClient_WithRetryAppliesMiddleware(t *testing.T) {
	client, err := NewClient(
		WithRetry(3, time.Millisecond, 2*time.Millisecond),
	)
	require.NoError(t, err)

	api := &testAPIClient{failUntil: 2}
	client.messages = service.NewMessageService(api, &noopCDNClient{}, client.contextTokens, client.middleware)
	client.SetContextToken("user-1", "ctx-token")

	err = client.SendText(context.Background(), "user-1", "hello")
	require.NoError(t, err)
	assert.Equal(t, 3, api.sendCalls)
}

func TestNewClient_WithRateLimitAppliesMiddleware(t *testing.T) {
	client, err := NewClient(
		WithRateLimit(50, 1),
	)
	require.NoError(t, err)

	api := &testAPIClient{}
	client.messages = service.NewMessageService(api, &noopCDNClient{}, client.contextTokens, client.middleware)
	client.SetContextToken("user-1", "ctx-token")

	start := time.Now()
	require.NoError(t, client.SendText(context.Background(), "user-1", "first"))
	require.NoError(t, client.SendText(context.Background(), "user-1", "second"))

	assert.Equal(t, 2, api.sendCalls)
	assert.GreaterOrEqual(t, time.Since(start), 18*time.Millisecond)
}

func TestClearTokenClearsInMemoryState(t *testing.T) {
	store := login.NewMemoryTokenStore()
	client, err := NewClient(WithTokenStore(store))
	require.NoError(t, err)

	require.NoError(t, store.Save(login.DefaultAccountID, &login.TokenInfo{Token: "stored"}))

	client.SetToken("live-token", "", login.DefaultAccountID, "user-1")
	client.SetContextToken("user-1", "ctx-token")

	client.clearToken(context.Background())

	loaded, err := store.Load(login.DefaultAccountID)
	require.NoError(t, err)
	assert.Nil(t, loaded)
	assert.False(t, client.IsLoggedIn())
	assert.Nil(t, client.CurrentUser())
	assert.Equal(t, "", client.GetContextToken("user-1"))
}

func TestRunDispatchesDisconnectedEvent(t *testing.T) {
	client, err := NewClient()
	require.NoError(t, err)

	client.SetToken("live-token", "", login.DefaultAccountID, "user-1")

	disconnected := make(chan struct{}, 1)
	client.Events().Subscribe(event.EventTypeDisconnected, func(ctx context.Context, e *event.Event) error {
		select {
		case disconnected <- struct{}{}:
		default:
		}
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = client.Run(ctx, nil)
	require.ErrorIs(t, err, context.Canceled)

	select {
	case <-disconnected:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected disconnected event")
	}
}

func TestRun_BacksOffAfterPollErrors(t *testing.T) {
	var (
		mu           sync.Mutex
		requestTimes []time.Time
		requestCount int
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestCount++
		requestTimes = append(requestTimes, time.Now())
		count := requestCount
		mu.Unlock()

		if r.URL.Path != "/ilink/bot/getupdates" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")

		if count <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"errmsg":"temporary failure"}`))
			return
		}

		_ = json.NewEncoder(w).Encode(ilink.GetUpdatesResponse{})
		cancel()
	}))
	defer server.Close()

	client, err := NewClient(
		WithBaseURL(server.URL),
		WithPollErrorBackoff(20*time.Millisecond, 50*time.Millisecond),
	)
	require.NoError(t, err)

	client.SetToken("live-token", server.URL, login.DefaultAccountID, "user-1")

	err = client.Run(ctx, nil)
	require.ErrorIs(t, err, context.Canceled)

	mu.Lock()
	times := append([]time.Time(nil), requestTimes...)
	mu.Unlock()

	require.Len(t, times, 3)
	assert.GreaterOrEqual(t, times[1].Sub(times[0]), 15*time.Millisecond)
	assert.GreaterOrEqual(t, times[2].Sub(times[1]), 30*time.Millisecond)
}

func TestClose_MakesActiveOperationsFail(t *testing.T) {
	client, err := NewClient()
	require.NoError(t, err)

	client.SetContextToken("user-1", "ctx-token")
	require.NoError(t, client.Close())

	err = client.Run(context.Background(), nil)
	require.ErrorIs(t, err, ErrClientClosed)

	err = client.SendText(context.Background(), "user-1", "hello")
	require.ErrorIs(t, err, ErrClientClosed)

	_, err = client.Login(context.Background(), nil)
	require.ErrorIs(t, err, ErrClientClosed)

	err = client.Logout(context.Background())
	require.ErrorIs(t, err, ErrClientClosed)
}
