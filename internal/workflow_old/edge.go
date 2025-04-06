package workflow

// Edge represents a connection between nodes
type Edge struct {
	FromNodeID string
	ToNodeID   string
	Condition  func(interface{}) bool // Optional condition for edge traversal
}
