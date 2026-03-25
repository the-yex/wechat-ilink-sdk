package plugin

import (
	"context"
	"fmt"
	"sync"

	"github.com/the-yex/wechat-ilink-sdk/ilink"
)

// Registry manages registered plugins.
type Registry struct {
	mu          sync.RWMutex
	plugins     map[string]Plugin
	pluginOrder []string // Keep insertion order for deterministic iteration
	sdk         SDK
}

// NewRegistry creates a new plugin registry.
func NewRegistry(sdk SDK) *Registry {
	return &Registry{
		plugins:     make(map[string]Plugin),
		pluginOrder: make([]string, 0),
		sdk:         sdk,
	}
}

// Register adds a plugin to the registry.
func (r *Registry) Register(p Plugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := p.Name()
	if _, exists := r.plugins[name]; exists {
		return fmt.Errorf("plugin %s already registered", name)
	}

	r.plugins[name] = p
	r.pluginOrder = append(r.pluginOrder, name)
	return nil
}

// Initialize initializes all registered plugins.
func (r *Registry) Initialize(ctx context.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, name := range r.pluginOrder {
		p := r.plugins[name]
		if err := p.Initialize(ctx, r.sdk); err != nil {
			return fmt.Errorf("initialize plugin %s: %w", name, err)
		}
	}
	return nil
}

// OnMessage calls OnMessage on all plugins in registration order.
func (r *Registry) OnMessage(ctx context.Context, msg *ilink.Message) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, name := range r.pluginOrder {
		p := r.plugins[name]
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

	for _, name := range r.pluginOrder {
		p := r.plugins[name]
		p.OnError(ctx, err)
	}
}

// Get returns a plugin by name.
func (r *Registry) Get(name string) (Plugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.plugins[name]
	return p, ok
}

// All returns all registered plugins.
func (r *Registry) All() []Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugins := make([]Plugin, 0, len(r.plugins))
	for _, p := range r.plugins {
		plugins = append(plugins, p)
	}
	return plugins
}
