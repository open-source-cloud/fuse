package workflow

// NodeProvider defines the basic interface for a NodeProvider
type NodeProvider interface {
	ID() string
	Nodes() []Node
	GetNode(id string) (Node, error)
}
