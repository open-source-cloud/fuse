package workflow

type NodeProvider interface {
	ID() string
	Nodes() []Node
}
