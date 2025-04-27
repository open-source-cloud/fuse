// Package graph provides a graph interface
package graph

type (
	// EdgeCondition represents a condition for an edge
	EdgeCondition struct {
		Name  string
		Value any
	}

	// Edge describes a graph's edge
	Edge interface {
		ID() string
		IsConditional() bool
		Condition() *EdgeCondition
		From() Node
		To() Node
	}
)
