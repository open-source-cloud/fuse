package workflow

// Workflow represents a complete workflow definition
type Workflow struct {
	ID          string
	Name        string
	Description string
	Nodes       []Node
	Edges       []Edge
}
