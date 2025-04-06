package workflow

type State interface {
	AddProvider(provider NodeProvider) error
	AddSchema(schema Schema) error
	AddNodeSpec(node NodeSpec) error
}
