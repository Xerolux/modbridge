package plugin

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
)

var (
	// ErrPluginNotFound is returned when a plugin is not found.
	ErrPluginNotFound = errors.New("plugin not found")
	// ErrPluginAlreadyRegistered is returned when a plugin is already registered.
	ErrPluginAlreadyRegistered = errors.New("plugin already registered")
)

// Protocol defines the interface for custom protocol plugins.
type Protocol interface {
	// Name returns the protocol name.
	Name() string

	// Version returns the protocol version.
	Version() string

	// Encode encodes data for transmission.
	Encode(data []byte) ([]byte, error)

	// Decode decodes received data.
	Decode(data []byte) ([]byte, error)

	// Validate validates a frame.
	Validate(frame []byte) error
}

// Transformer defines the interface for data transformation plugins.
type Transformer interface {
	// Name returns the transformer name.
	Name() string

	// Transform transforms data.
	Transform(data []byte, params map[string]interface{}) ([]byte, error)
}

// Middleware defines the interface for request/response middleware plugins.
type Middleware interface {
	// Name returns the middleware name.
	Name() string

	// Priority returns the middleware priority (lower = higher priority).
	Priority() int

	// ProcessRequest processes an incoming request.
	ProcessRequest(ctx context.Context, req []byte, conn net.Conn) ([]byte, error)

	// ProcessResponse processes an outgoing response.
	ProcessResponse(ctx context.Context, resp []byte, conn net.Conn) ([]byte, error)
}

// Handler defines the interface for custom request handlers.
type Handler interface {
	// Name returns the handler name.
	Name() string

	// CanHandle returns true if this handler can process the request.
	CanHandle(req []byte) bool

	// Handle handles the request and returns a response.
	Handle(ctx context.Context, req []byte) ([]byte, error)
}

// Plugin represents a loaded plugin.
type Plugin struct {
	Name        string
	Version     string
	Description string
	Author      string

	// Plugin instances
	Protocol    Protocol
	Transformer Transformer
	Middleware  Middleware
	Handler     Handler

	// Metadata
	Metadata map[string]interface{}

	// Lifecycle hooks
	OnLoad   func() error
	OnUnload func() error
}

// Manager manages plugins.
type Manager struct {
	mu           sync.RWMutex
	protocols    map[string]*Plugin
	transformers map[string]*Plugin
	middlewares  []*Plugin // Sorted by priority
	handlers     map[string]*Plugin
}

// NewManager creates a new plugin manager.
func NewManager() *Manager {
	return &Manager{
		protocols:    make(map[string]*Plugin),
		transformers: make(map[string]*Plugin),
		middlewares:  make([]*Plugin, 0),
		handlers:     make(map[string]*Plugin),
	}
}

// Register registers a plugin.
func (m *Manager) Register(plugin *Plugin) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Call OnLoad hook if present
	if plugin.OnLoad != nil {
		if err := plugin.OnLoad(); err != nil {
			return fmt.Errorf("plugin load failed: %w", err)
		}
	}

	// Register based on plugin type
	if plugin.Protocol != nil {
		name := plugin.Protocol.Name()
		if _, exists := m.protocols[name]; exists {
			return ErrPluginAlreadyRegistered
		}
		m.protocols[name] = plugin
	}

	if plugin.Transformer != nil {
		name := plugin.Transformer.Name()
		if _, exists := m.transformers[name]; exists {
			return ErrPluginAlreadyRegistered
		}
		m.transformers[name] = plugin
	}

	if plugin.Middleware != nil {
		m.middlewares = append(m.middlewares, plugin)
		m.sortMiddlewares()
	}

	if plugin.Handler != nil {
		name := plugin.Handler.Name()
		if _, exists := m.handlers[name]; exists {
			return ErrPluginAlreadyRegistered
		}
		m.handlers[name] = plugin
	}

	return nil
}

// Unregister unregisters a plugin.
func (m *Manager) Unregister(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Find and unregister from all categories
	var plugin *Plugin

	if p, exists := m.protocols[name]; exists {
		plugin = p
		delete(m.protocols, name)
	}

	if p, exists := m.transformers[name]; exists {
		plugin = p
		delete(m.transformers, name)
	}

	if p, exists := m.handlers[name]; exists {
		plugin = p
		delete(m.handlers, name)
	}

	// Remove from middlewares
	for i, p := range m.middlewares {
		if p.Name == name {
			plugin = p
			m.middlewares = append(m.middlewares[:i], m.middlewares[i+1:]...)
			break
		}
	}

	if plugin == nil {
		return ErrPluginNotFound
	}

	// Call OnUnload hook if present
	if plugin.OnUnload != nil {
		if err := plugin.OnUnload(); err != nil {
			return fmt.Errorf("plugin unload failed: %w", err)
		}
	}

	return nil
}

// GetProtocol gets a protocol plugin by name.
func (m *Manager) GetProtocol(name string) (Protocol, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plugin, exists := m.protocols[name]
	if !exists {
		return nil, ErrPluginNotFound
	}

	return plugin.Protocol, nil
}

// GetTransformer gets a transformer plugin by name.
func (m *Manager) GetTransformer(name string) (Transformer, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plugin, exists := m.transformers[name]
	if !exists {
		return nil, ErrPluginNotFound
	}

	return plugin.Transformer, nil
}

// GetMiddlewares returns all middleware plugins in priority order.
func (m *Manager) GetMiddlewares() []Middleware {
	m.mu.RLock()
	defer m.mu.RUnlock()

	middlewares := make([]Middleware, len(m.middlewares))
	for i, plugin := range m.middlewares {
		middlewares[i] = plugin.Middleware
	}

	return middlewares
}

// GetHandler finds a handler that can process the request.
func (m *Manager) GetHandler(req []byte) (Handler, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, plugin := range m.handlers {
		if plugin.Handler.CanHandle(req) {
			return plugin.Handler, nil
		}
	}

	return nil, ErrPluginNotFound
}

// ListPlugins returns all registered plugins.
func (m *Manager) ListPlugins() []PluginInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	seen := make(map[string]bool)
	plugins := make([]PluginInfo, 0)

	addPlugin := func(plugin *Plugin, pluginType string) {
		if seen[plugin.Name] {
			return
		}
		seen[plugin.Name] = true

		plugins = append(plugins, PluginInfo{
			Name:        plugin.Name,
			Version:     plugin.Version,
			Description: plugin.Description,
			Author:      plugin.Author,
			Type:        pluginType,
		})
	}

	for _, plugin := range m.protocols {
		addPlugin(plugin, "protocol")
	}

	for _, plugin := range m.transformers {
		addPlugin(plugin, "transformer")
	}

	for _, plugin := range m.middlewares {
		addPlugin(plugin, "middleware")
	}

	for _, plugin := range m.handlers {
		addPlugin(plugin, "handler")
	}

	return plugins
}

// sortMiddlewares sorts middlewares by priority.
func (m *Manager) sortMiddlewares() {
	// Simple bubble sort (fine for small lists)
	for i := 0; i < len(m.middlewares); i++ {
		for j := i + 1; j < len(m.middlewares); j++ {
			if m.middlewares[j].Middleware.Priority() < m.middlewares[i].Middleware.Priority() {
				m.middlewares[i], m.middlewares[j] = m.middlewares[j], m.middlewares[i]
			}
		}
	}
}

// PluginInfo holds plugin information.
type PluginInfo struct {
	Name        string
	Version     string
	Description string
	Author      string
	Type        string
}
