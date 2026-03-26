// Package event provides an event system for the iLink SDK.
package event

import (
	"context"
	"log"
	"sync"
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
type Dispatcher struct {
	mu       sync.RWMutex
	handlers map[EventType][]Handler
}

// NewDispatcher creates a new event dispatcher.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		handlers: make(map[EventType][]Handler),
	}
}

// Subscribe registers a handler for an event type.
func (d *Dispatcher) Subscribe(eventType EventType, handler Handler) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.handlers[eventType] = append(d.handlers[eventType], handler)
}

// Unsubscribe removes all handlers for an event type.
func (d *Dispatcher) Unsubscribe(eventType EventType) {
	d.mu.Lock()
	defer d.mu.Unlock()

	delete(d.handlers, eventType)
}

// Dispatch dispatches an event to all registered handlers.
// Handlers are called asynchronously to avoid blocking.
// Panics in handlers are recovered and logged.
func (d *Dispatcher) Dispatch(ctx context.Context, event *Event) {
	d.mu.RLock()
	handlers := make([]Handler, len(d.handlers[event.Type]))
	copy(handlers, d.handlers[event.Type])
	d.mu.RUnlock()

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
func (d *Dispatcher) DispatchSync(ctx context.Context, event *Event) error {
	d.mu.RLock()
	handlers := d.handlers[event.Type]
	d.mu.RUnlock()

	for _, h := range handlers {
		if err := h(ctx, event); err != nil {
			return err
		}
	}
	return nil
}