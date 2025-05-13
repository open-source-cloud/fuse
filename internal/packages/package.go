package packages

type Package interface {
	ID() string
	Functions() []FunctionSpec
	GetFunction(id string) (FunctionSpec, error)
}
