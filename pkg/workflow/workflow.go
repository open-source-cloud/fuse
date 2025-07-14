package workflow

import "github.com/open-source-cloud/fuse/pkg/uuid"

// ID defines a Workflow ID type
type ID string

func (id ID) String() string {
	return string(id)
}

// NewID generates a new Workflow ID
func NewID() ID {
	return ID(uuid.V7())
}
