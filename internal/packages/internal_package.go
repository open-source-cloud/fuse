package packages

import (
	"fmt"
)

type internalFunctionPackage struct {
	id        string
	functions map[string]FunctionSpec
}

func NewInternal(id string, functions []FunctionSpec) Package {
	functionsMap := make(map[string]FunctionSpec)
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

func (f *internalFunctionPackage) Functions() []FunctionSpec {
	functions := make([]FunctionSpec, 0, len(f.functions))
	for _, function := range f.functions {
		functions = append(functions, function)
	}
	return functions
}

func (f *internalFunctionPackage) GetFunction(id string) (FunctionSpec, error) {
	if function, ok := f.functions[id]; ok {
		return function, nil
	}
	return nil, fmt.Errorf("function %s not found", id)
}
