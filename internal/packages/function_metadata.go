package packages

import (
	"github.com/open-source-cloud/fuse/pkg/transport"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

type (
	// FunctionMetadata metadata for a registered function
	FunctionMetadata struct {
		Transport transport.Type         `json:"transport"`
		Input     FunctionInputMetadata  `json:"input"`
		Output    FunctionOutputMetadata `json:"output"`
	}

	// FunctionInputMetadata input metadata for a registered function
	FunctionInputMetadata struct {
		CustomParameters bool                                `json:"customParameters"`
		Parameters       map[string]workflow.ParameterSchema `json:"parameters"`
		Edges            FunctionInputEdgeMetadata           `json:"edges"`
	}

	// FunctionInputEdgeMetadata input's edge metadata for a registered function
	FunctionInputEdgeMetadata struct {
		Count      int                                 `json:"count"`
		Parameters map[string]workflow.ParameterSchema `json:"parameters"`
	}

	// FunctionOutputMetadata output metadata for a registered function
	FunctionOutputMetadata struct {
		Parameters             map[string]workflow.ParameterSchema   `json:"parameters"`
		ConditionalOutput      bool                                  `json:"conditionalOutput"`
		ConditionalOutputField string                                `json:"conditionalOutputField"`
		Edges                  map[string]FunctionOutputEdgeMetadata `json:"edges"`
	}

	// FunctionOutputEdgeMetadata output's edge metadata for a registered function
	FunctionOutputEdgeMetadata struct {
		Name            string                              `json:"name"`
		ConditionalEdge workflow.ConditionalEdgeMetadata    `json:"conditionalEdge"`
		Count           int                                 `json:"count"`
		Parameters      map[string]workflow.ParameterSchema `json:"parameters"`
	}
)
