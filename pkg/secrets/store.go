package secrets

import (
	"context"
	"errors"
	"strings"
)

// ErrSecretNotFound is returned when a secret cannot be resolved.
var ErrSecretNotFound = errors.New("secret not found")

// Scope identifies the resolution context for a secret. Environment is the
// primary scoping dimension; WorkflowID is available for future per-workflow
// overrides (backends may ignore it).
type Scope struct {
	Environment string
	WorkflowID  string
}

// SecretStore resolves secrets by name within a scope (read-only).
type SecretStore interface {
	Resolve(ctx context.Context, scope Scope, name string) (SecretValue, error)
}

// ManagedSecretStore additionally supports administration. The memory and
// encrypted-Postgres backends implement it; read-only backends (e.g. an external
// manager) implement only SecretStore.
type ManagedSecretStore interface {
	SecretStore
	Set(ctx context.Context, scope Scope, name, value string) error
	List(ctx context.Context, environment string) ([]string, error)
	Delete(ctx context.Context, scope Scope, name string) error
}

// Resolver is the narrow capability the workflow engine depends on: resolve a
// secret by name for the running workflow. The environment is bound in, so the
// engine never deals with scoping or the store directly.
type Resolver interface {
	Resolve(ctx context.Context, workflowID, name string) (SecretValue, error)
}

// NewResolver binds a SecretStore + environment into a Resolver.
func NewResolver(store SecretStore, environment string) Resolver {
	return &scopedResolver{store: store, environment: environment}
}

type scopedResolver struct {
	store       SecretStore
	environment string
}

func (r *scopedResolver) Resolve(ctx context.Context, workflowID, name string) (SecretValue, error) {
	return r.store.Resolve(ctx, Scope{Environment: r.environment, WorkflowID: workflowID}, strings.TrimSpace(name))
}
