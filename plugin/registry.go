package plugin

import (
	"context"
	"fmt"
	"sync"

	"github.com/the-yex/wechat-ilink-sdk/ilink"
)

// Registry manages registered plugins.
// Uses a simple slice for optimal iteration performance (CPU cache friendly).
// Plugin count is typically small (<10), so O(n) lookup is acceptable.
type Registry struct {
	mu          sync.RWMutex
	plugins     []Plugin
	initialized map[string]struct{}
	sdk         SDK
}

// NewRegistry creates a new plugin registry.
func NewRegistry(sdk SDK) *Registry {
	return &Registry{
		plugins:     make([]Plugin, 0),
		initialized: make(map[string]struct{}),
		sdk:         sdk,
	}
}

// Register adds a plugin to the registry.
// Returns error if a plugin with the same name already exists.
func (r *Registry) Register(p Plugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := p.Name()
	// Check for duplicate (O(n), but n is small)
	for _, existing := range r.plugins {
		if existing.Name() == name {
			return fmt.Errorf("plugin %s already registered", name)
		}
	}

	r.plugins = append(r.plugins, p)
	return nil
}

// Initialize initializes all registered plugins in registration order.
func (r *Registry) Initialize(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, p := range r.plugins {
		if err := r.initializeLocked(ctx, p); err != nil {
			return fmt.Errorf("initialize plugin %s: %w", p.Name(), err)
		}
	}
	return nil
}

// InitializeOne initializes a registered plugin if it has not been initialized yet.
// Repeated calls are safe and will not re-run plugin initialization.
func (r *Registry) InitializeOne(ctx context.Context, p Plugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.initializeLocked(ctx, p)
}

// OnMessage calls OnMessage on all plugins in registration order.
func (r *Registry) OnMessage(ctx context.Context, msg *ilink.Message) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.plugins {
		if err := p.OnMessage(ctx, msg); err != nil {
			return err
		}
	}
	return nil
}

// OnError calls OnError on all plugins.
func (r *Registry) OnError(ctx context.Context, err error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.plugins {
		p.OnError(ctx, err)
	}
}

// Get returns a plugin by name.
// Returns nil, false if not found.
func (r *Registry) Get(name string) (Plugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.plugins {
		if p.Name() == name {
			return p, true
		}
	}
	return nil, false
}

// All returns all registered plugins.
func (r *Registry) All() []Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make([]Plugin, len(r.plugins))
	copy(result, r.plugins)
	return result
}

func (r *Registry) initializeLocked(ctx context.Context, p Plugin) error {
	name := p.Name()
	if _, ok := r.initialized[name]; ok {
		return nil
	}
	if err := p.Initialize(ctx, r.sdk); err != nil {
		return err
	}
	r.initialized[name] = struct{}{}
	return nil
}
