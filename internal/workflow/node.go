package workflow

type NodeSpec struct {
	ID string
}

type Node struct {
	ID   string
	Spec *NodeSpec
}

type Edge struct {
	From *Node
	To   *Node
}

type NodeInstance interface {
	ID() string
}

type DefaultNodeInstance struct{}
