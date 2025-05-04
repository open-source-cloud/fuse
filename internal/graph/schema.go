package graph

import (
	"encoding/json"
	"gopkg.in/yaml.v3"
)

type (
	// RootSchema represents the structure containing the name and graph definition for a particular configuration.
	RootSchema struct {
		Name  string `json:"name" yaml:"name"`
		Graph Schema `json:"graph" yaml:"graph"`
	}
	// Schema represents a structure containing an ID and a list of NodeSchema elements.
	Schema struct {
		ID    string       `json:"id" yaml:"id"`
		Nodes []NodeSchema `json:"nodes" yaml:"nodes"`
	}
	// NodeSchema represents a node structure with an ID, associated package, and optional input configurations.
	NodeSchema struct {
		ID      string            `json:"id" yaml:"id"`
		Package PackageSchema     `json:"package" yaml:"package"`
		Config  *NodeConfigSchema `json:"config,omitempty" yaml:"config,omitempty"`
	}
	// NodeConfigSchema represents the configuration schema for a node containing a list of input schemas.
	NodeConfigSchema struct {
		Inputs []NodeInputSchema `json:"inputs,omitempty" yaml:"inputs,omitempty"`
	}
	// NodeInputSchema represents the schema for a single input to a node, including its source, origin, and mapping details.
	NodeInputSchema struct {
		Source  string `json:"source" yaml:"source"`
		Origin  any    `json:"origin" yaml:"origin"`
		Mapping string `json:"mapping" yaml:"mapping"`
	}
	// EdgeSchema represents the schema for an edge in a graph with an ID, source, destination, and optional metadata.
	EdgeSchema struct {
		ID          string             `json:"id" yaml:"id"`
		From        string             `json:"from" yaml:"from"`
		To          string             `json:"to" yaml:"to"`
		Conditional *ConditionalSchema `json:"conditional,omitempty" yaml:"conditional,omitempty"`
	}
	// ConditionalSchema represents a conditional configuration with a name and its associated value.
	ConditionalSchema struct {
		Name  string `json:"name" yaml:"name"`
		Value any    `json:"value" yaml:"value"`
	}
	// PackageSchema specifies a package's registry and associated function.
	PackageSchema struct {
		Registry string `json:"registry" yaml:"registry"`
		Function string `json:"function" yaml:"function"`
	}
)

func CreateSchemaFromYaml(yamlSpec []byte) (*RootSchema, error) {
	var schema RootSchema
	err := yaml.Unmarshal(yamlSpec, &schemaDef)
	if err != nil {
		return nil, err
	}
	return &schema, nil
}

func CreateSchemaFromJSON(jsonSpec []byte) (*RootSchema, error) {
	var schema RootSchema
	err := json.Unmarshal(jsonSpec, &schema)
	if err != nil {
	}
	return &schema, nil
}
