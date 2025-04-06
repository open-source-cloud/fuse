package workflow

type NodeProviderSpec struct {
	ID    string
	Nodes []NodeSpec
}

type NodeProvider interface {
	ID() string
	Nodes() []NodeSpec
}

type DefaultNodeProvider struct {
	id    string
	nodes []NodeSpec
}

func NewDefaultNodeProvider(spec NodeProviderSpec) *DefaultNodeProvider {
	return &DefaultNodeProvider{
		id:    spec.ID,
		nodes: spec.Nodes,
	}
}

func (d *DefaultNodeProvider) ID() string {
	return d.id
}
func (d *DefaultNodeProvider) Nodes() []NodeSpec {
	return d.nodes
}
