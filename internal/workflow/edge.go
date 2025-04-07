package workflow

import (
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

type DefaultEdge struct {
	id          string
	dataMapping workflow.DataMapping
}

func NewDefaultEdge(id string, dataMapping workflow.DataMapping) *DefaultEdge {
	return &DefaultEdge{
		id:          id,
		dataMapping: dataMapping,
	}
}

func (e *DefaultEdge) ID() string {
	return e.id
}

func (e *DefaultEdge) DataMapping() workflow.DataMapping {
	return e.dataMapping
}
