// Package graph provides a graph interface
package graph

// Edge describes a graph's edge
type (
	EdgeCondition struct {
		Name  string
		Value any
	}

	Edge interface {
		ID() string
		IsConditional() bool
		Condition() *EdgeCondition
		From() Node
		To() Node
	}
)
