package bot

import "sync"

// Registry holds registered modules.
type Registry struct {
	mu      sync.RWMutex
	modules []Module
}

// NewRegistry creates a new module registry.
func NewRegistry() *Registry {
	return &Registry{
		modules: make([]Module, 0),
	}
}

// Register adds a module to the registry.
func (r *Registry) Register(m Module) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.modules = append(r.modules, m)
}

// Modules returns a snapshot of all registered modules.
func (r *Registry) Modules() []Module {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make([]Module, len(r.modules))
	copy(result, r.modules)
	return result
}

// Global registry instance for module self-registration via init()
var globalRegistry = NewRegistry()

// Register adds a module to the global registry.
// This is typically called from module init() functions.
func Register(m Module) {
	globalRegistry.Register(m)
}

// Modules returns all modules from the global registry.
func Modules() []Module {
	return globalRegistry.Modules()
}

// ResetGlobalRegistry resets the global registry.
// This is intended for testing purposes only.
func ResetGlobalRegistry() {
	globalRegistry = NewRegistry()
}
