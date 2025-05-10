package workflow

// Package defines the basic interface for a NodeProvider
type Package interface {
	ID() string
	Functions() []Function
	GetFunction(id string) (Function, error)
}
