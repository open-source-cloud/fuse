// Package graph provides a graph interface
package graph

// Edge describes a graph's edge
type Edge interface {
	ID() string
	From() Node
	To() Node
}
