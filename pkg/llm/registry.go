package llm

import (
	"context"
	"fmt"
)

// ErrProviderNotFound is returned when a requested provider is not registered.
var ErrProviderNotFound = fmt.Errorf("llm provider not found")

// ErrNoDefaultProvider is returned when no default provider is configured.
var ErrNoDefaultProvider = fmt.Errorf("no default llm provider configured")

// ProviderFactory builds a Provider for a given resolution environment (ADR-0031). It is called
// per execution so providers whose API key / base URL are secret references resolve against the
// running workflow's environment; factories for fully-static config return a prebuilt singleton.
type ProviderFactory func(ctx context.Context, environment string) (Provider, error)

// Registry resolves configured LLM providers by name, scoped to an environment.
type Registry interface {
	// Get returns the provider registered under name, built for the given environment.
	Get(ctx context.Context, environment, name string) (Provider, error)
	// Default returns the configured default provider, built for the given environment.
	Default(ctx context.Context, environment string) (Provider, error)
	// List returns the names of all registered providers.
	List() []string
}

// memoryRegistry is the default in-memory Registry implementation.
type memoryRegistry struct {
	factories   map[string]ProviderFactory
	defaultName string
}

// NewRegistry builds a Registry from per-provider factories. defaultName selects the provider
// returned by Default; it may be empty if no default is desired.
func NewRegistry(factories map[string]ProviderFactory, defaultName string) Registry {
	f := make(map[string]ProviderFactory, len(factories))
	for name, factory := range factories {
		f[name] = factory
	}
	return &memoryRegistry{factories: f, defaultName: defaultName}
}

// NewStaticRegistry builds a Registry from already-constructed providers, ignoring the
// environment. It is the fast path for fully-static configuration and the convenience
// constructor for tests.
func NewStaticRegistry(providers map[string]Provider, defaultName string) Registry {
	factories := make(map[string]ProviderFactory, len(providers))
	for name, prov := range providers {
		p := prov
		factories[name] = func(_ context.Context, _ string) (Provider, error) { return p, nil }
	}
	return NewRegistry(factories, defaultName)
}

// Get returns the provider registered under name, built for the given environment.
func (r *memoryRegistry) Get(ctx context.Context, environment, name string) (Provider, error) {
	factory, ok := r.factories[name]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrProviderNotFound, name)
	}
	return factory(ctx, environment)
}

// Default returns the configured default provider, built for the given environment.
func (r *memoryRegistry) Default(ctx context.Context, environment string) (Provider, error) {
	if r.defaultName == "" {
		return nil, ErrNoDefaultProvider
	}
	return r.Get(ctx, environment, r.defaultName)
}

// List returns the names of all registered providers.
func (r *memoryRegistry) List() []string {
	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}
	return names
}
