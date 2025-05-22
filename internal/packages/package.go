package packages

// Package defines the interface of a Package
type Package interface {
	ID() string
	Functions() []FunctionSpec
	GetFunction(id string) (FunctionSpec, error)
}
