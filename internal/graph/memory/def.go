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
	// EdgeRefDefConditional is the definition of a conditional edge
	EdgeRefDefConditional struct {
		Name  string `json:"name" yaml:"name"`
		Value any    `json:"value" yaml:"value"`
	}
	// EdgeRefDef represents an edge with a potential condition
	EdgeRefDef struct {
		NodeID      string                 `json:"node" yaml:"node"`
		Conditional *EdgeRefDefConditional `json:"conditional,omitempty" yaml:"conditional,omitempty"`
	}
	// EdgeDef is the definition of an edge
	EdgeDef struct {
		ID             string       `json:"id" yaml:"id"`
		NodeParentRefs []EdgeRefDef `json:"references,omitempty" yaml:"references,omitempty"`
	}
	// InputDef is the definition of an input
	InputDef struct {
		Source  string `json:"source" yaml:"source"`
		Origin  any    `json:"origin" yaml:"origin"`
		Mapping string `json:"mapping" yaml:"mapping"`
	}
)

// CreateSchemaFromYaml creates a schema from a YAML file
func CreateSchemaFromYaml(yamlSpec []byte, providerRegistry *providers.Registry) (*SchemaDef, graph.Graph, error) {
	var schemaDef SchemaDef
	err := yaml.Unmarshal(yamlSpec, &schemaDef)
	if err != nil {
		return nil, nil, err
	}
	graphDef, err := createGraphDef(schemaDef.Graph, providerRegistry)
	if err != nil {
		return nil, nil, err
	}
	return &schemaDef, graphDef, nil
}

// CreateSchemaFromJSON creates a schema from a JSON file
func CreateSchemaFromJSON(jsonSpec []byte, providerRegistry *providers.Registry) (*SchemaDef, graph.Graph, error) {
	var schemaDef SchemaDef
	err := json.Unmarshal(jsonSpec, &schemaDef)
	if err != nil {
		return nil, nil, err
	}
	graphDef, err := createGraphDef(schemaDef.Graph, providerRegistry)
	if err != nil {
		return nil, nil, err
	}
	return &schemaDef, graphDef, nil
}

// createGraphDef creates a graph from a graph definition
func createGraphDef(graphDef GraphDef, providerRegistry *providers.Registry) (graph.Graph, error) {
	log.Info().Msgf("Create newGraph from definition: %s", graphDef.ID)

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
			log.Debug().Msgf("rootNode.Inputs.input.Origin: %s", input.Origin)
			log.Debug().Msgf("rootNode.Inputs.input.Mapping: %s", input.Mapping)
			rootNodeConfig.AddInputMapping(input.Source, input.Origin, input.Mapping)
		}
	}

	log.Debug().Msgf("root node created")
	log.Debug().Msgf("rootNode.ID: %s", rootNode.ID())

	newGraph := NewGraph(rootNode)

	// We're going to use this map to reference the nodes by their ID
	// Because the nodeRefId is different from the newGraph.Node.ID() -> (fmt.Sprintf("%s/%s", workflowNode.ID(), uuid))
	nodesMapRef := make(map[string]*Node)

	for _, nodeDef := range graphDef.Nodes {
		log.Debug().Msgf("creating node: %s", nodeDef.ID)
		log.Debug().Msgf("nodeDef.ID: %s", nodeDef.ID)
		log.Debug().Msgf("nodeDef.Provider.ID: %s", nodeDef.Provider.ID)
		log.Debug().Msgf("nodeDef.Provider.NodeID: %s", nodeDef.Provider.NodeID)
		log.Debug().Msgf("nodeDef.Edge.ID: %s", nodeDef.Edge.ID)
		log.Debug().Msgf("nodeDef.Edge.NodeParentRefs: %v", nodeDef.Edge.NodeParentRefs)

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
				log.Debug().Msgf("nodeDef.Inputs.input.Origin: %s", input.Origin)
				log.Debug().Msgf("nodeDef.Inputs.input.Mapping: %s", input.Mapping)
				nodeConfig.AddInputMapping(input.Source, input.Origin, input.Mapping)
			}
		}

		node := NewNode(nodeDef.ID, nodeWorkflow, nodeConfig)
		nodesMapRef[nodeDef.ID] = node

		switch len(nodeDef.Edge.NodeParentRefs) {
		// No parent, add to root as parent
		case 0:
			err = addNodeToParent(newGraph, node, rootNode, nodeDef.Edge.ID, nil)
			if err != nil {
				log.Error().Msgf("error adding node %s to root node %s: %s", nodeDef.ID, rootNode.ID(), err)
				return nil, err
			}
		// One parent, add to parent
		case 1:
			nodeParentRef := nodeDef.Edge.NodeParentRefs[0]
			nodeRef := nodesMapRef[nodeParentRef.NodeID]
			if nodeRef == nil {
				log.Error().Msgf("node %s not found", nodeParentRef.NodeID)
				return nil, fmt.Errorf("node %s not found", nodeParentRef.NodeID)
			}
			err = addNodeToParent(newGraph, node, nodeRef, nodeDef.Edge.ID, nodeParentRef.Conditional)
			if err != nil {
				log.Error().Msgf("error adding node %s to parent node %s: %s", nodeDef.ID, nodesMapRef[nodeParentRef.NodeID].ID(), err)
				return nil, err
			}
		// Multiple parents, add to all parents
		default:
			// Convert the incoming nodeDef.Edge.NodeParentRefs to the newGraph.Node.ID()
			// because the nodeDef.Edge.NodeParentRefs is the nodeDef.ID,
			// and we need to use the newGraph.Node.ID() to add the node to the newGraph (e.g., fuse.io/workflows/internal/logic/rand/logic-rand-1)
			parents := make([]graph.ParentNodeWithCondition, len(nodeDef.Edge.NodeParentRefs))
			for i, nodeParentRef := range nodeDef.Edge.NodeParentRefs {
				parent := nodesMapRef[nodeParentRef.NodeID]
				if parent == nil {
					log.Error().Msgf("node %s not found", nodeParentRef.NodeID)
					return nil, fmt.Errorf("node %s not found", nodeParentRef.NodeID)
				}
				condition := getEdgeCondition(nodeParentRef.Conditional)
				parents[i] = graph.ParentNodeWithCondition{NodeID: parent.ID(), Condition: condition}
			}
			err = addNodeToMultipleParents(newGraph, node, parents, nodeDef.Edge.ID)
			if err != nil {
				log.Error().Msgf("error adding node %s to parent nodes %v: %s", nodeDef.ID, nodeDef.Edge.NodeParentRefs, err)
				return nil, err
			}
		}
	}

	return newGraph, nil
}

// addNodeToParent adds a node to a parent node
func addNodeToParent(graphDef graph.Graph, node *Node, parent *Node, edgeID string, conditional *EdgeRefDefConditional) error {
	log.Debug().Msgf("adding node %s to parent node %s", node.ID(), parent.ID())
	err := graphDef.AddNode(parent.ID(), edgeID, node, getEdgeCondition(conditional))
	if err != nil {
		log.Error().Msgf("error adding node %s to parent node %s: %s", node.ID(), parent.ID(), err)
		return err
	}
	return nil
}

// addNodeToMultipleParents adds a node to multiple parents
func addNodeToMultipleParents(graphDef graph.Graph, node *Node, parents []graph.ParentNodeWithCondition, edgeID string) error {
	log.Debug().Msgf("adding node %s to parent nodes %v", node.ID(), parents)
	err := graphDef.AddNodeMultipleParents(parents, edgeID, node)
	if err != nil {
		log.Error().Msgf("error adding node %s to parent nodes %v: %s", node.ID(), parents, err)
		return err
	}
	return nil
}

func getEdgeCondition(conditional *EdgeRefDefConditional) *graph.EdgeCondition {
	if conditional != nil {
		return &graph.EdgeCondition{
			Name:  conditional.Name,
			Value: conditional.Value,
		}
	}
	return nil
}
