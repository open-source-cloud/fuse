package llm

import "fmt"

// ErrProviderNotFound is returned when a requested provider is not registered.
var ErrProviderNotFound = fmt.Errorf("llm provider not found")

// ErrNoDefaultProvider is returned when no default provider is configured.
var ErrNoDefaultProvider = fmt.Errorf("no default llm provider configured")

// Registry resolves configured LLM providers by name.
type Registry interface {
	// Get returns the provider registered under name.
	Get(name string) (Provider, error)
	// Default returns the configured default provider.
	Default() (Provider, error)
	// List returns the names of all registered providers.
	List() []string
}

// memoryRegistry is the default in-memory Registry implementation.
type memoryRegistry struct {
	providers   map[string]Provider
	defaultName string
}

// NewRegistry builds a Registry from the given providers. defaultName selects
// the provider returned by Default; it may be empty if no default is desired.
func NewRegistry(providers map[string]Provider, defaultName string) Registry {
	p := make(map[string]Provider, len(providers))
	for name, prov := range providers {
		p[name] = prov
	}
	return &memoryRegistry{providers: p, defaultName: defaultName}
}

// Get returns the provider registered under name.
func (r *memoryRegistry) Get(name string) (Provider, error) {
	prov, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrProviderNotFound, name)
	}
	return prov, nil
}

// Default returns the configured default provider.
func (r *memoryRegistry) Default() (Provider, error) {
	if r.defaultName == "" {
		return nil, ErrNoDefaultProvider
	}
	return r.Get(r.defaultName)
}

// List returns the names of all registered providers.
func (r *memoryRegistry) List() []string {
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}
