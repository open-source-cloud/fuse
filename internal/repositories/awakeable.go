package repositories

import (
	"errors"

	workflow "github.com/open-source-cloud/fuse/internal/workflow"
)

// ErrAwakeableNotFound is returned when an awakeable is not found
var ErrAwakeableNotFound = errors.New("awakeable not found")

// AwakeableRepository defines the interface for awakeable persistence
type AwakeableRepository interface {
	Save(awakeable *workflow.Awakeable) error
	FindByID(id string) (*workflow.Awakeable, error)
	FindPending(workflowID string) ([]*workflow.Awakeable, error)
	Resolve(id string, result map[string]any) error
}
