package workflow

import (
	"github.com/open-source-cloud/fuse/internal/packages"
	"strings"
)

func NewGraphFactory(packageRegistry packages.Registry) *GraphFactory {
	return &GraphFactory{
		packageRegistry: packageRegistry,
	}
}

func NewGraphFactoryWithoutMetadata() *GraphFactory {
	return &GraphFactory{
		packageRegistry: nil,
	}
}

type GraphFactory struct{
	packageRegistry packages.Registry
}

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