// Package schema represents the graph schema for a workflow
package schema

import (
	"encoding/json"
	"gopkg.in/yaml.v3"
)

// Graph represents a data structure containing nodes and edges, identified by a unique ID and optionally named.
type Graph struct {
	ID    string  `json:"id" yaml:"id"`
	Name  string  `json:"name" yaml:"name"`
	Nodes []*Node `json:"nodes" yaml:"nodes"`
	Edges []*Edge `json:"edges" yaml:"edges"`
}

// FromYaml parses a YAML specification and constructs a Graph object. Returns error if parsing fails.
func FromYaml(yamlSpec []byte) (*Graph, error) {
	var graph Graph
	err := yaml.Unmarshal(yamlSpec, &graph)
	if err != nil {
		return nil, err
	}
	return &graph, nil
}

// FromJSON parses a JSON representation of a Graph and returns a Graph instance or an error if parsing fails.
func FromJSON(jsonSpec []byte) (*Graph, error) {
	var graph Graph
	err := json.Unmarshal(jsonSpec, &graph)
	if err != nil {
		return nil, err
	}
	return &graph, nil
}
