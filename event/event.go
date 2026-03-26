// Package event provides an event system for the iLink SDK.
package event

import (
	"context"
	"log"
	"sync"

	"github.com/the-yex/wechat-ilink-sdk/internal/t"
)

// EventType represents the type of event.
type EventType int

const (
	EventTypeMessage EventType = iota + 1
	EventTypeLogin
	EventTypeError
	EventTypeSessionExpired
	EventTypeConnected
	EventTypeDisconnected
)

// Event represents an SDK event.
type Event struct {
	Type    EventType
	Data    interface{}
	Context context.Context
}

// Handler handles events.
type Handler func(ctx context.Context, event *Event) error

// Dispatcher manages event handlers.
// Uses t.Map for lock-free reads during dispatch.
type Dispatcher struct {
	handlers *t.Map[EventType, []Handler]
	mu       sync.Mutex // Only protects Subscribe (low-frequency operation)
}

// NewDispatcher creates a new event dispatcher.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		handlers: t.New[EventType, []Handler](),
	}
}

// Subscribe registers a handler for an event type.
// This is a low-frequency operation (typically called during initialization).
func (d *Dispatcher) Subscribe(eventType EventType, handler Handler) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Load existing handlers
	existing, _ := d.handlers.Load(eventType)

	// Create new slice with the handler appended (immutable pattern)
	newHandlers := make([]Handler, len(existing)+1)
	copy(newHandlers, existing)
	newHandlers[len(existing)] = handler

	// Store the new slice
	d.handlers.Store(eventType, newHandlers)
}

// Unsubscribe removes all handlers for an event type.
func (d *Dispatcher) Unsubscribe(eventType EventType) {
	d.handlers.Delete(eventType)
}

// Dispatch dispatches an event to all registered handlers.
// Handlers are called asynchronously to avoid blocking.
// Panics in handlers are recovered and logged.
// This is a lock-free operation.
func (d *Dispatcher) Dispatch(ctx context.Context, event *Event) {
	handlers, ok := d.handlers.Load(event.Type)
	if !ok {
		return
	}

	// Call handlers asynchronously with panic recovery
	for _, h := range handlers {
		go func(handler Handler) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[event] handler panic recovered: %v", r)
				}
			}()
			if err := handler(ctx, event); err != nil {
				log.Printf("[event] handler error: %v", err)
			}
		}(h)
	}
}

// DispatchSync dispatches an event synchronously.
// Returns the first error encountered.
// This is a lock-free operation.
func (d *Dispatcher) DispatchSync(ctx context.Context, event *Event) error {
	handlers, ok := d.handlers.Load(event.Type)
	if !ok {
		return nil
	}

	for _, h := range handlers {
		if err := h(ctx, event); err != nil {
			return err
		}
	}
	return nil
}