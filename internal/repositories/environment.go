package repositories

import (
	"errors"

	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// ErrEnvironmentNotFound is returned when an environment is not found.
var ErrEnvironmentNotFound = errors.New("environment not found")

type (
	// EnvironmentRepository defines the interface of an environment repository (ADR-0031).
	EnvironmentRepository interface {
		FindByID(name string) (*workflow.Environment, error)
		FindAll() ([]*workflow.Environment, error)
		Save(env *workflow.Environment) error
		Delete(name string) error
	}
)
