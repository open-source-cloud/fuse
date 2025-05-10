package packages

import (
	"fmt"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

type internalFunctionPackage struct {
	id        string
	functions map[string]workflow.Function
}

func NewInternal(id string, functions []workflow.Function) workflow.Package {
	functionsMap := make(map[string]workflow.Function)
	for _, function := range functions {
		functionsMap[function.ID()] = function
	}

	return &internalFunctionPackage{
		id:        id,
		functions: functionsMap,
	}
}

func (f *internalFunctionPackage) ID() string {
	return f.id
}

func (f *internalFunctionPackage) Functions() []workflow.Function {
	functions := make([]workflow.Function, 0, len(f.functions))
	for _, function := range f.functions {
		functions = append(functions, function)
	}
	return functions
}

func (f *internalFunctionPackage) GetFunction(id string) (workflow.Function, error) {
	if function, ok := f.functions[id]; ok {
		return function, nil
	}
	return nil, fmt.Errorf("function %s not found", id)
}
