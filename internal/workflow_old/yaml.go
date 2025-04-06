package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// YAMLWorkflow represents a workflow definition in YAML
type YAMLWorkflow struct {
	ID          string     `yaml:"id"`
	Name        string     `yaml:"name"`
	Description string     `yaml:"description"`
	Nodes       []YAMLNode `yaml:"nodes"`
	Edges       []YAMLEdge `yaml:"edges"`
}

// YAMLNode represents a node definition in YAML
type YAMLNode struct {
	ID       string                 `yaml:"id"`
	Type     string                 `yaml:"type"`     // e.g., "string_processor", "if", "switch"
	Provider string                 `yaml:"provider"` // e.g., "strproc", "logic"
	Config   map[string]interface{} `yaml:"config"`
}

// YAMLEdge represents an edge definition in YAML
type YAMLEdge struct {
	From      string                 `yaml:"from"`
	To        string                 `yaml:"to"`
	Condition map[string]interface{} `yaml:"condition,omitempty"`
}

// LoadWorkflowFromYAML loads a workflow from a YAML file
func LoadWorkflowFromYAML(filename string) (*YAMLWorkflow, error) {
	// Ensure the file path is safe
	if !filepath.IsAbs(filename) {
		filename = filepath.Clean(filename)
	}
	if strings.Contains(filename, "..") {
		return nil, fmt.Errorf("unsafe file path: %s", filename)
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow file: %w", err)
	}

	var wf YAMLWorkflow
	if err := yaml.Unmarshal(data, &wf); err != nil {
		return nil, fmt.Errorf("failed to parse workflow YAML: %v", err)
	}

	return &wf, nil
}

// ConvertYAMLToWorkflow converts a YAML workflow definition to a Workflow
func ConvertYAMLToWorkflow(ywf *YAMLWorkflow, providers map[string]NodeProvider) (*Workflow, error) {
	wf := &Workflow{
		ID:          ywf.ID,
		Name:        ywf.Name,
		Description: ywf.Description,
		Nodes:       make([]Node, len(ywf.Nodes)),
		Edges:       make([]Edge, len(ywf.Edges)),
	}

	// Create nodes
	nodeMap := make(map[string]Node)
	for i, yn := range ywf.Nodes {
		provider, exists := providers[yn.Provider]
		if !exists {
			return nil, fmt.Errorf("provider %s not found", yn.Provider)
		}

		node, err := provider.CreateNode(yn.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to create node %s: %v", yn.ID, err)
		}

		wf.Nodes[i] = node
		nodeMap[yn.ID] = node
	}

	// Create edges
	for i, ye := range ywf.Edges {
		fromNode, exists := nodeMap[ye.From]
		if !exists {
			return nil, fmt.Errorf("from node %s not found", ye.From)
		}

		toNode, exists := nodeMap[ye.To]
		if !exists {
			return nil, fmt.Errorf("to node %s not found", ye.To)
		}

		wf.Edges[i] = Edge{
			FromNodeID: fromNode.ID(),
			ToNodeID:   toNode.ID(),
		}
	}

	return wf, nil
}
