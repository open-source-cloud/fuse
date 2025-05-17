package workflow

// GraphSchema represents a data structure containing nodes and edges, identified by a unique ID and optionally named.
type GraphSchema struct {
	ID    string        `json:"id" yaml:"id"`
	Name  string        `json:"name" yaml:"name"`
	Nodes []*NodeSchema `json:"nodes" yaml:"nodes"`
	Edges []*EdgeSchema `json:"edges" yaml:"edges"`
}
