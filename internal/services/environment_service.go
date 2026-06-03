package services

import (
	"github.com/open-source-cloud/fuse/internal/repositories"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

type (
	// EnvironmentService manages the environments registry (ADR-0031) and validates the
	// environment a workflow trigger requests.
	EnvironmentService interface {
		FindAll() ([]*workflow.Environment, error)
		FindByID(name string) (*workflow.Environment, error)
		Save(env *workflow.Environment) (*workflow.Environment, error)
		Delete(name string) error
		// IsValid reports whether name is a declared environment. The default environment is
		// always valid even if the registry has not been seeded.
		IsValid(name string) bool
	}

	// DefaultEnvironmentService is the default EnvironmentService implementation.
	DefaultEnvironmentService struct {
		repo repositories.EnvironmentRepository
	}
)

// NewEnvironmentService returns a new EnvironmentService.
func NewEnvironmentService(repo repositories.EnvironmentRepository) EnvironmentService {
	return &DefaultEnvironmentService{repo: repo}
}

// FindAll returns all declared environments.
func (s *DefaultEnvironmentService) FindAll() ([]*workflow.Environment, error) {
	return s.repo.FindAll()
}

// FindByID returns a single environment by name.
func (s *DefaultEnvironmentService) FindByID(name string) (*workflow.Environment, error) {
	return s.repo.FindByID(name)
}

// Save validates and upserts an environment.
func (s *DefaultEnvironmentService) Save(env *workflow.Environment) (*workflow.Environment, error) {
	if err := env.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Save(env); err != nil {
		return nil, err
	}
	return env, nil
}

// Delete removes an environment by name.
func (s *DefaultEnvironmentService) Delete(name string) error {
	return s.repo.Delete(name)
}

// IsValid reports whether name is a declared environment (the default is always valid).
func (s *DefaultEnvironmentService) IsValid(name string) bool {
	if name == workflow.DefaultEnvironmentName {
		return true
	}
	_, err := s.repo.FindByID(name)
	return err == nil
}
