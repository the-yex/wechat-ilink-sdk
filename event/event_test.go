package event

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDispatcher_CloseWaitsForAsyncHandlers(t *testing.T) {
	dispatcher := NewDispatcher()
	started := make(chan struct{})
	release := make(chan struct{})
	finished := make(chan struct{})

	dispatcher.Subscribe(EventTypeConnected, func(ctx context.Context, event *Event) error {
		close(started)
		<-release
		close(finished)
		return nil
	})

	dispatcher.Dispatch(context.Background(), &Event{Type: EventTypeConnected})

	select {
	case <-started:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected handler to start")
	}

	closed := make(chan struct{})
	go func() {
		dispatcher.Close()
		close(closed)
	}()

	select {
	case <-closed:
		t.Fatal("dispatcher closed before handler finished")
	case <-time.After(20 * time.Millisecond):
	}

	close(release)

	select {
	case <-finished:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected handler to finish")
	}

	select {
	case <-closed:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected dispatcher close to wait for handler")
	}
}

func TestDispatcher_ClosePreventsNewDispatches(t *testing.T) {
	dispatcher := NewDispatcher()
	calls := 0

	dispatcher.Subscribe(EventTypeError, func(ctx context.Context, event *Event) error {
		calls++
		return nil
	})

	dispatcher.Close()
	dispatcher.Dispatch(context.Background(), &Event{Type: EventTypeError})
	dispatcher.Wait()

	assert.Equal(t, 0, calls)
	require.ErrorIs(t, dispatcher.DispatchSync(context.Background(), &Event{Type: EventTypeError}), ErrDispatcherClosed)
}
