package packages

import (
	"fmt"

	"github.com/open-source-cloud/fuse/internal/actors/actor"
	"github.com/open-source-cloud/fuse/internal/packages/transport"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// NewLoadedPackage creates a new LoadedPackage
func NewLoadedPackage(id string, functions map[string]*LoadedFunction) *LoadedPackage {
	return &LoadedPackage{
		ID:        id,
		Functions: functions,
	}
}

// LoadedPackage defines the interface of a LoadedPackage
type LoadedPackage struct {
	ID        string                     `json:"id"`
	Functions map[string]*LoadedFunction `json:"functions"`
}

// GetFunctionMetadata gets function metadata from FunctionID
func (p *LoadedPackage) GetFunctionMetadata(functionID string) (*FunctionMetadata, error) {
	function, exists := p.Functions[functionID]
	if !exists {
		return nil, fmt.Errorf("function %s not found", functionID)
	}
	return function.Metadata, nil
}

// ExecuteFunction executes function based on FunctionID
func (p *LoadedPackage) ExecuteFunction(handle actor.Handle, functionID string, execInfo *workflow.ExecutionInfo) (workflow.FunctionResult, error) {
	function, exists := p.Functions[functionID]
	if !exists {
		return workflow.FunctionResult{}, fmt.Errorf("function %s not found", functionID)
	}

	return function.Transport.Execute(handle, execInfo)
}

// MapToRegistryPackage converts from pkg/packages.Package into internal/packages.LoadedPackage
func MapToRegistryPackage(pkg *workflow.Package) *LoadedPackage {
	functions := make(map[string]*LoadedFunction, len(pkg.Functions))
	for _, function := range pkg.Functions {
		functionID := fmt.Sprintf("%s/%s", pkg.ID, function.ID)

		m := function.Metadata
		metadata := &FunctionMetadata{
			Transport: transport.Internal,
			Input: FunctionInputMetadata{
				CustomParameters: m.Input.CustomParameters,
				Parameters:       make(map[string]workflow.ParameterSchema, len(m.Input.Parameters)),
				Edges: FunctionInputEdgeMetadata{
					Count:      m.Input.Edges.Count,
					Parameters: make(map[string]workflow.ParameterSchema, len(m.Input.Edges.Parameters)),
				},
			},
			Output: FunctionOutputMetadata{
				Parameters:             make(map[string]workflow.ParameterSchema, len(m.Output.Parameters)),
				ConditionalOutput:      m.Output.ConditionalOutput,
				ConditionalOutputField: m.Output.ConditionalOutputField,
				Edges:                  make(map[string]FunctionOutputEdgeMetadata, len(m.Output.Edges)),
			},
		}

		// input maps
		for _, param := range m.Input.Parameters {
			metadata.Input.Parameters[param.Name] = param
		}
		for _, edge := range m.Input.Edges.Parameters {
			metadata.Input.Edges.Parameters[edge.Name] = edge
		}

		// output maps
		for _, param := range m.Output.Parameters {
			metadata.Output.Parameters[param.Name] = param
		}
		for _, edge := range m.Output.Edges {
			outputEdge := FunctionOutputEdgeMetadata{
				Name:            edge.Name,
				ConditionalEdge: edge.ConditionalEdge,
				Count:           edge.Count,
				Parameters:      make(map[string]workflow.ParameterSchema, len(edge.Parameters)),
			}
			for _, param := range edge.Parameters {
				outputEdge.Parameters[param.Name] = param
			}
			metadata.Output.Edges[edge.Name] = outputEdge
		}

		if function.Metadata.Transport == transport.Internal {
			functions[functionID] = NewLoadedInternalFunction(
				functionID,
				metadata,
				function.Function,
			)
		} else {
			functions[functionID] = NewLoadedFunction(
				functionID,
				metadata,
			)
		}
	}
	return NewLoadedPackage(pkg.ID, functions)
}
