package services

import (
	"context"
	"errors"
	"sort"

	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/pkg/secrets"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// ErrReadOnlySecretStore is returned when credential values are written against a read-only
// secret backend. Credential management requires a ManagedSecretStore (memory or postgres).
var ErrReadOnlySecretStore = errors.New("the configured SECRETS_DRIVER is read-only; credential values require driver=memory or driver=postgres")

type (
	// CredentialService manages credential metadata and their per-environment field values
	// (ADR-0031 Option B). Metadata lives in the CredentialRepository; values live in the
	// SecretStore at cred/<id>/<field>. Values are never returned by reads.
	CredentialService interface {
		FindAll() ([]*workflow.Credential, error)
		FindByID(id string) (*workflow.Credential, error)
		Save(cred *workflow.Credential, fieldValues map[string]string, environment string) (*workflow.Credential, error)
		Delete(id string, environment string) error
		Resolve(ctx context.Context, environment, id, field string) (secrets.SecretValue, error)
	}

	// DefaultCredentialService is the default CredentialService implementation.
	DefaultCredentialService struct {
		repo  repositories.CredentialRepository
		store secrets.SecretStore
	}
)

// NewCredentialService returns a new CredentialService.
func NewCredentialService(repo repositories.CredentialRepository, store secrets.SecretStore) CredentialService {
	return &DefaultCredentialService{repo: repo, store: store}
}

// FindAll returns all credential metadata (never values).
func (s *DefaultCredentialService) FindAll() ([]*workflow.Credential, error) {
	return s.repo.FindAll()
}

// FindByID returns a single credential's metadata (never values).
func (s *DefaultCredentialService) FindByID(id string) (*workflow.Credential, error) {
	return s.repo.FindByID(id)
}

// Save validates and persists the credential metadata, then writes each field value to the
// SecretStore at cred/<id>/<field> in the given environment. The credential's Fields are the
// union of any previously-recorded field names and the provided value keys, so metadata tracks
// every field that has a value (and incremental single-field updates don't drop others).
func (s *DefaultCredentialService) Save(cred *workflow.Credential, fieldValues map[string]string, environment string) (*workflow.Credential, error) {
	existing := make([]string, 0)
	if prev, err := s.repo.FindByID(cred.ID); err == nil {
		existing = prev.Fields
	}
	cred.Fields = unionSorted(existing, keys(fieldValues))
	if err := cred.Validate(); err != nil {
		return nil, err
	}

	managed, ok := s.store.(secrets.ManagedSecretStore)
	if !ok {
		return nil, ErrReadOnlySecretStore
	}

	if err := s.repo.Save(cred); err != nil {
		return nil, err
	}

	for field, value := range fieldValues {
		if err := managed.Set(context.Background(), secrets.Scope{Environment: environment}, secrets.CredentialSecretName(cred.ID, field), value); err != nil {
			return nil, err
		}
	}
	return cred, nil
}

// Delete removes the credential's field secrets in the given environment, then its metadata.
func (s *DefaultCredentialService) Delete(id string, environment string) error {
	cred, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}

	if managed, ok := s.store.(secrets.ManagedSecretStore); ok {
		for _, field := range cred.Fields {
			if delErr := managed.Delete(context.Background(), secrets.Scope{Environment: environment}, secrets.CredentialSecretName(id, field)); delErr != nil {
				return delErr
			}
		}
	}
	return s.repo.Delete(id)
}

// Resolve returns a credential field's value for an environment as a redacted SecretValue.
func (s *DefaultCredentialService) Resolve(ctx context.Context, environment, id, field string) (secrets.SecretValue, error) {
	return s.store.Resolve(ctx, secrets.Scope{Environment: environment}, secrets.CredentialSecretName(id, field))
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// unionSorted returns the sorted, de-duplicated union of two field-name slices.
func unionSorted(a, b []string) []string {
	set := make(map[string]struct{}, len(a)+len(b))
	for _, v := range a {
		set[v] = struct{}{}
	}
	for _, v := range b {
		set[v] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for v := range set {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}
