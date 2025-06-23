package workflow

import (
	"github.com/open-source-cloud/fuse/internal/packages"
	"strings"
)

// NewGraphFactory creates a new GraphFactory factory
func NewGraphFactory(packageRegistry packages.Registry) *GraphFactory {
	return &GraphFactory{
		packageRegistry: packageRegistry,
	}
}

// NewGraphFactoryWithoutMetadata creates a new GraphFactory factory without any packages - mainly for mermaid gen
func NewGraphFactoryWithoutMetadata() *GraphFactory {
	return &GraphFactory{
		packageRegistry: nil,
	}
}

// GraphFactory graph factory
type GraphFactory struct {
	packageRegistry packages.Registry
}

// NewGraphFromJSON creates a new Graph from a JSON schema
func (f *GraphFactory) NewGraphFromJSON(jsonSpec []byte) (*Graph, error) {
	graph, err := newGraphFromJSON(jsonSpec)
	if err != nil {
		return nil, err
	}
	if err := f.populateMetadata(graph); err != nil {
		return nil, err
	}
	return graph, nil
}

// NewGraphFromYAML creates a new Graph from a YAML schema
func (f *GraphFactory) NewGraphFromYAML(yamlSpec []byte) (*Graph, error) {
	graph, err := newGraphFromYAML(yamlSpec)
	if err != nil {
		return nil, err
	}
	if err := f.populateMetadata(graph); err != nil {
		return nil, err
	}
	return graph, nil
}

// NewGraphFromSchema returns a new Graph from GraphSchema
func (f *GraphFactory) NewGraphFromSchema(schema *GraphSchema) (*Graph, error) {
	graph, err := newGraphFromSchema(schema)
	if err != nil {
		return nil, err
	}
	if err := f.populateMetadata(graph); err != nil {
		return nil, err
	}
	return graph, nil
}

func (f *GraphFactory) populateMetadata(graph *Graph) error {
	if f.packageRegistry == nil {
		return nil
	}

	for _, node := range graph.nodes {
		lastIndexOfSlash := strings.LastIndex(node.schema.Function, "/")
		pkgID := node.schema.Function[:lastIndexOfSlash]
		pkg, err := f.packageRegistry.Get(pkgID)
		if err != nil {
			return err
		}
		pkgFn, err := pkg.GetFunction(node.schema.Function)
		if err != nil {
			return err
		}
		node.functionMetadata = pkgFn.Metadata()
	}
	return nil
}
