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
	// WorkflowNodeDef is the definition of a workflow node
	ProviderDef struct {
		NodeProviderID string `json:"nodeProviderId" yaml:"nodeProviderId"`
		NodeID         string `json:"nodeId" yaml:"nodeId"`
	}
	// EdgeDef is the definition of an edge
	EdgeDef struct {
		ID        string   `json:"id" yaml:"id"`
		ParentID  string   `json:"parentId,omitempty" yaml:"parentId,omitempty"`
		ParentIDs []string `json:"parentIds,omitempty" yaml:"parentIds,omitempty"`
	}
	// InputDef is the definition of an input
	InputDef struct {
		Source    string `json:"source" yaml:"source"`
		ParamName string `json:"paramName" yaml:"paramName"`
		Mapping   string `json:"mapping" yaml:"mapping"`
	}
)

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

func createGraphDef(graphDef GraphDef, providerRegistry *providers.Registry) (graph.Graph, error) {
	log.Info().Msgf("createGraphDef: %s", graphDef.ID)

	// TODO: Maybe move this to a store function?
	// We're going to refactor this when introduce the node provider registry store
	rootNodeProvider, err := providerRegistry.Get(graphDef.Root.Provider.NodeProviderID)
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
		log.Debug().Msgf("nodeDef.Provider.NodeProviderID: %s", nodeDef.Provider.NodeProviderID)
		log.Debug().Msgf("nodeDef.Provider.NodeID: %s", nodeDef.Provider.NodeID)
		log.Debug().Msgf("nodeDef.Edge.ID: %s", nodeDef.Edge.ID)
		log.Debug().Msgf("nodeDef.Edge.ParentID: %s", nodeDef.Edge.ParentID)
		log.Debug().Msgf("nodeDef.Edge.ParentIDs: %v", nodeDef.Edge.ParentIDs)

		nodeProvider, err := providerRegistry.Get(nodeDef.Provider.NodeProviderID)
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

		// If there are multiple parent IDs, add the node to the parents
		if len(nodeDef.Edge.ParentIDs) > 0 {
			var parentNodes []string
			for _, parentID := range nodeDef.Edge.ParentIDs {
				parentNodes = append(parentNodes, nodesMapRef[parentID].ID())
			}
			log.Debug().Msgf("parentNodes: %v", parentNodes)
			if len(parentNodes) == 0 {
				log.Error().Msgf("no parent nodes found for node %s", nodeDef.ID)
				return nil, fmt.Errorf("no parent nodes found for node %s", nodeDef.ID)
			}
			log.Debug().Msgf("adding node %s to parent nodes %v", nodeDef.ID, parentNodes)
			err = graph.AddNodeMultipleParents(parentNodes, nodeDef.Edge.ID, node)
			if err != nil {
				log.Error().Msgf("error adding node %s to parent nodes %v: %s", nodeDef.ID, parentNodes, err)
				return nil, err
			}
			continue
		}

		// If there is a parent ID, add the node to the parent
		if nodeDef.Edge.ParentID != "" {
			parentNode := nodesMapRef[nodeDef.Edge.ParentID]
			if parentNode == nil {
				log.Error().Msgf("parent node %s not found for node %s", nodeDef.Edge.ParentID, nodeDef.ID)
				return nil, fmt.Errorf("parent node %s not found for node %s", nodeDef.Edge.ParentID, nodeDef.ID)
			}
			log.Debug().Msgf("parentNode: %s", parentNode.ID())
			err = graph.AddNode(parentNode.ID(), nodeDef.Edge.ID, node)
			if err != nil {
				log.Error().Msgf("error adding node %s to parent node %s: %s", nodeDef.ID, nodeDef.Edge.ParentID, err)
				return nil, err
			}
			continue
		}

		// If there is no parent ID, add the node to the root node
		err = graph.AddNode(rootNode.ID(), nodeDef.Edge.ID, node)
		if err != nil {
			log.Error().Msgf("error adding node %s to root node %s: %s", nodeDef.ID, rootNode.ID(), err)
			return nil, err
		}
	}

	return graph, nil
}
