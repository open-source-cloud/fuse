package workflow

// Node represents an executable Node and it's metadata
type Node interface {
	ID() string
	Metadata() NodeMetadata
	Execute(input *NodeInput) (NodeResult, error)
}
