package workflow

type (
	// Edge defines an Edge for workflow graphs
	Edge struct {
		id     string
		schema *EdgeSchema
		from   *Node
		to     *Node
	}
)

// NewEdge creates and returns a new Edge with the specified from and to nodes.
func newEdge(id string, from *Node, to *Node, schema *EdgeSchema) *Edge {
	return &Edge{
		id:     id,
		schema: schema,
		from:   from,
		to:     to,
	}
}

// ID returns the Edge ID
func (e *Edge) ID() string {
	return e.id
}

// IsConditional returns true if this Edge has a conditional
func (e *Edge) IsConditional() bool {
	return e.schema.Conditional != nil
}

// Condition returns the Edge conditional
func (e *Edge) Condition() *EdgeCondition {
	return e.schema.Conditional
}

// Input returns the Edge input mappings
func (e *Edge) Input() []InputMapping {
	return e.schema.Input
}

// From returns the Edge from Node
func (e *Edge) From() *Node {
	return e.from
}

// To return the Edge to Node
func (e *Edge) To() *Node {
	return e.to
}
