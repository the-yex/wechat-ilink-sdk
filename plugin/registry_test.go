package plugin

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/the-yex/wechat-ilink-sdk/ilink"
)

// Mock SDK implementation
type mockSDK struct {
	lastText string
	lastTo   string
}

func (m *mockSDK) SendText(ctx context.Context, toUserID, text string) error {
	m.lastTo = toUserID
	m.lastText = text
	return nil
}

func (m *mockSDK) SendMessage(ctx context.Context, req *ilink.SendMessageRequest) error {
	return nil
}

func (m *mockSDK) UploadMedia(ctx context.Context, req interface{}) (interface{}, error) {
	return nil, nil
}

func (m *mockSDK) DownloadMedia(ctx context.Context, req interface{}) ([]byte, error) {
	return nil, nil
}

type mockPlugin struct {
	name        string
	initErr     error
	onMsgErr    error
	onMsgCalled bool
}

func (p *mockPlugin) Name() string { return p.name }

func (p *mockPlugin) Initialize(ctx context.Context, sdk SDK) error {
	return p.initErr
}

func (p *mockPlugin) OnMessage(ctx context.Context, msg *ilink.Message) error {
	p.onMsgCalled = true
	return p.onMsgErr
}

func (p *mockPlugin) OnError(ctx context.Context, err error) {}

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry(nil)

	t.Run("success", func(t *testing.T) {
		p := &mockPlugin{name: "test-plugin"}
		err := registry.Register(p)
		require.NoError(t, err)
	})

	t.Run("duplicate", func(t *testing.T) {
		p1 := &mockPlugin{name: "duplicate"}
		p2 := &mockPlugin{name: "duplicate"}

		_ = registry.Register(p1)
		err := registry.Register(p2)
		require.Error(t, err)
	})
}

func TestRegistry_Initialize(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		registry := NewRegistry(nil)
		p := &mockPlugin{name: "test"}
		_ = registry.Register(p)

		err := registry.Initialize(context.Background())
		require.NoError(t, err)
	})

	t.Run("init error", func(t *testing.T) {
		registry := NewRegistry(nil)
		p := &mockPlugin{name: "test", initErr: assert.AnError}
		_ = registry.Register(p)

		err := registry.Initialize(context.Background())
		require.Error(t, err)
	})
}

func TestRegistry_OnMessage(t *testing.T) {
	t.Run("all success", func(t *testing.T) {
		registry := NewRegistry(nil)
		p1 := &mockPlugin{name: "p1"}
		p2 := &mockPlugin{name: "p2"}
		_ = registry.Register(p1)
		_ = registry.Register(p2)

		err := registry.OnMessage(context.Background(), &ilink.Message{})

		require.NoError(t, err)
		assert.True(t, p1.onMsgCalled)
		assert.True(t, p2.onMsgCalled)
	})

	t.Run("plugin error stops chain", func(t *testing.T) {
		registry := NewRegistry(nil)
		p1 := &mockPlugin{name: "p1", onMsgErr: assert.AnError}
		p2 := &mockPlugin{name: "p2"}
		_ = registry.Register(p1)
		_ = registry.Register(p2)

		err := registry.OnMessage(context.Background(), &ilink.Message{})

		require.Error(t, err)
		assert.True(t, p1.onMsgCalled)
		assert.False(t, p2.onMsgCalled)
	})
}

func TestRegistry_OnError(t *testing.T) {
	registry := NewRegistry(nil)
	p := &mockPlugin{name: "test"}
	_ = registry.Register(p)

	// OnError should not panic
	registry.OnError(context.Background(), assert.AnError)
}