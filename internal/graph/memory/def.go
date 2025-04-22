package memory

import (
	"encoding/json"
	"fmt"

	"github.com/open-source-cloud/fuse/internal/providers"
	"github.com/open-source-cloud/fuse/pkg/graph"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

type (
	// SchemaDef is the definition of a schema
	SchemaDef struct {
		Name  string   `json:"name" yaml:"name"`
		Graph GraphDef `json:"graph" yaml:"graph"`
	}
	// GraphDef is the definition of a graph
	GraphDef struct {
		ID    string    `json:"id" yaml:"id"`
		Root  NodeDef   `json:"root" yaml:"root"`
		Nodes []NodeDef `json:"nodes" yaml:"nodes"`
	}
	// NodeDef is the definition of a node
	NodeDef struct {
		ID       string      `json:"id" yaml:"id"`
		Provider ProviderDef `json:"provider" yaml:"provider"`
		Edge     EdgeDef     `json:"edge,omitempty" yaml:"edge,omitempty"`
		Inputs   []InputDef  `json:"inputs,omitempty" yaml:"inputs,omitempty"`
	}
	// ProviderDef is the definition of a workflow node
	ProviderDef struct {
		ID     string `json:"package" yaml:"package"`
		NodeID string `json:"node" yaml:"node"`
	}
	// EdgeDef is the definition of an edge
	EdgeDef struct {
		ID             string   `json:"id" yaml:"id"`
		NodeParentRefs []string `json:"references,omitempty" yaml:"references,omitempty"`
	}
	// InputDef is the definition of an input
	InputDef struct {
		Source    string `json:"source" yaml:"source"`
		ParamName string `json:"paramName" yaml:"paramName"`
		Mapping   string `json:"mapping" yaml:"mapping"`
	}
)

// CreateSchemaFromYaml creates a schema from a yaml file
func CreateSchemaFromYaml(yamlSpec []byte, providerRegistry *providers.Registry) (*SchemaDef, graph.Graph, error) {
	var schemaDef SchemaDef
	err := yaml.Unmarshal(yamlSpec, &schemaDef)
	if err != nil {
		return nil, nil, err
	}
	graph, err := createGraphDef(schemaDef.Graph, providerRegistry)
	if err != nil {
		return nil, nil, err
	}
	return &schemaDef, graph, nil
}

// CreateSchemaFromJSON creates a schema from a json file
func CreateSchemaFromJSON(jsonSpec []byte, providerRegistry *providers.Registry) (*SchemaDef, graph.Graph, error) {
	var schemaDef SchemaDef
	err := json.Unmarshal(jsonSpec, &schemaDef)
	if err != nil {
		return nil, nil, err
	}
	graph, err := createGraphDef(schemaDef.Graph, providerRegistry)
	if err != nil {
		return nil, nil, err
	}
	return &schemaDef, graph, nil
}

// createGraphDef creates a graph from a graph definition
func createGraphDef(graphDef GraphDef, providerRegistry *providers.Registry) (graph.Graph, error) {
	log.Info().Msgf("Create graph from definition: %s", graphDef.ID)

	rootNodeProvider, err := providerRegistry.Get(graphDef.Root.Provider.ID)
	if err != nil {
		log.Error().Msgf("error getting root node provider: %s", err)
		return nil, err
	}

	rootWorkflowNode, err := rootNodeProvider.GetNode(graphDef.Root.Provider.NodeID)
	if err != nil {
		log.Error().Msgf("error getting root workflow node: %s", err)
		return nil, err
	}

	rootDef := graphDef.Root

	rootNodeConfig := NewNodeConfig()
	rootNode := NewNode(rootDef.ID, rootWorkflowNode, rootNodeConfig)

	if len(rootDef.Inputs) > 0 {
		for _, input := range rootDef.Inputs {
			log.Debug().Msgf("rootNode.Inputs.input.Source: %s", input.Source)
			log.Debug().Msgf("rootNode.Inputs.input.ParamName: %s", input.ParamName)
			log.Debug().Msgf("rootNode.Inputs.input.Mapping: %s", input.Mapping)
			rootNodeConfig.AddInputMapping(input.Source, input.ParamName, input.Mapping)
		}
	}

	log.Debug().Msgf("root node created")
	log.Debug().Msgf("rootNode.ID: %s", rootNode.ID())

	graph := NewGraph(rootNode)

	// We're going to use this map to reference the nodes by their ID
	// Because the nodeRefId is different from the graph.Node.ID() -> (fmt.Sprintf("%s/%s", workflowNode.ID(), uuid))
	nodesMapRef := make(map[string]*Node)

	for _, nodeDef := range graphDef.Nodes {
		log.Debug().Msgf("creating node: %s", nodeDef.ID)
		log.Debug().Msgf("nodeDef.ID: %s", nodeDef.ID)
		log.Debug().Msgf("nodeDef.Provider.ID: %s", nodeDef.Provider.ID)
		log.Debug().Msgf("nodeDef.Provider.NodeID: %s", nodeDef.Provider.NodeID)
		log.Debug().Msgf("nodeDef.Edge.ID: %s", nodeDef.Edge.ID)
		log.Debug().Msgf("nodeDef.Edge.NodeParentRefs: %s", nodeDef.Edge.NodeParentRefs)

		nodeProvider, err := providerRegistry.Get(nodeDef.Provider.ID)
		if err != nil {
			return nil, err
		}

		nodeWorkflow, err := nodeProvider.GetNode(nodeDef.Provider.NodeID)
		if err != nil {
			return nil, err
		}

		nodeConfig := NewNodeConfig()
		if len(nodeDef.Inputs) > 0 {
			for _, input := range nodeDef.Inputs {
				log.Debug().Msgf("nodeDef.Inputs.input.Source: %s", input.Source)
				log.Debug().Msgf("nodeDef.Inputs.input.ParamName: %s", input.ParamName)
				log.Debug().Msgf("nodeDef.Inputs.input.Mapping: %s", input.Mapping)
				nodeConfig.AddInputMapping(input.Source, input.ParamName, input.Mapping)
			}
		}

		node := NewNode(nodeDef.ID, nodeWorkflow, nodeConfig)
		nodesMapRef[nodeDef.ID] = node

		switch len(nodeDef.Edge.NodeParentRefs) {
		// No parent, add to root
		case 0:
			err = addNodeToRoot(graph, rootNode, node, nodeDef.Edge.ID)
			if err != nil {
				log.Error().Msgf("error adding node %s to root node %s: %s", nodeDef.ID, rootNode.ID(), err)
				return nil, err
			}
		// One parent, add to parent
		case 1:
			parentID := nodeDef.Edge.NodeParentRefs[0]
			nodeRef := nodesMapRef[parentID]
			if nodeRef == nil {
				log.Error().Msgf("node %s not found", parentID)
				return nil, fmt.Errorf("node %s not found", parentID)
			}
			err = addNodeToParent(graph, node, nodeRef, nodeDef.Edge.ID)
			if err != nil {
				log.Error().Msgf("error adding node %s to parent node %s: %s", nodeDef.ID, nodesMapRef[nodeDef.Edge.NodeParentRefs[0]].ID(), err)
				return nil, err
			}
		// Multiple parents, add to all parents
		default:
			// Convert the incoming nodeDef.Edge.NodeParentRefs to the graph.Node.ID()
			// because the nodeDef.Edge.NodeParentRefs is the nodeDef.ID
			// and we need to use the graph.Node.ID() to add the node to the graph (e.g fuse.io/workflows/internal/logic/rand/logic-rand-1)
			parents := make([]string, len(nodeDef.Edge.NodeParentRefs))
			for i, parentID := range nodeDef.Edge.NodeParentRefs {
				parent := nodesMapRef[parentID]
				if parent == nil {
					log.Error().Msgf("node %s not found", parentID)
					return nil, fmt.Errorf("node %s not found", parentID)
				}
				parents[i] = parent.ID()
			}
			err = addNodeToMultipleParents(graph, node, parents, nodeDef.Edge.ID)
			if err != nil {
				log.Error().Msgf("error adding node %s to parent nodes %v: %s", nodeDef.ID, nodeDef.Edge.NodeParentRefs, err)
				return nil, err
			}
		}
	}

	return graph, nil
}

// addNodeToParent adds a node to a parent node
func addNodeToParent(graph graph.Graph, node *Node, parent *Node, edgeID string) error {
	log.Debug().Msgf("adding node %s to parent node %s", node.ID(), parent.ID())
	err := graph.AddNode(parent.ID(), edgeID, node)
	if err != nil {
		log.Error().Msgf("error adding node %s to parent node %s: %s", node.ID(), parent.ID(), err)
		return err
	}
	return nil
}

// addNodeToMultipleParents adds a node to multiple parents
func addNodeToMultipleParents(graph graph.Graph, node *Node, parents []string, edgeID string) error {
	log.Debug().Msgf("adding node %s to parent nodes %v", node.ID(), parents)
	err := graph.AddNodeMultipleParents(parents, edgeID, node)
	if err != nil {
		log.Error().Msgf("error adding node %s to parent nodes %v: %s", node.ID(), parents, err)
		return err
	}
	return nil
}

// addNodeToRoot adds a node to the root node
func addNodeToRoot(graph graph.Graph, rootNode *Node, node *Node, edgeID string) error {
	log.Debug().Msgf("adding node %s to root node %s", node.ID(), rootNode.ID())
	err := graph.AddNode(rootNode.ID(), edgeID, node)
	if err != nil {
		log.Error().Msgf("error adding node %s to root node %s: %s", node.ID(), rootNode.ID(), err)
		return err
	}
	return nil
}
