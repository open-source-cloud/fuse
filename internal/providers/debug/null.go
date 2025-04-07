package debug

import (
	"fmt"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

type NullNode struct{}

func (n *NullNode) ID() string {
	return fmt.Sprintf("%s/null", debugProviderID)
}

func (n *NullNode) InputSchema() *workflow.DataSchema {
	return nil
}

func (n *NullNode) OutputSchemas(name string) *workflow.DataSchema {
	return nil
}

func (n *NullNode) Execute(input map[string]any) (interface{}, error) {
	return nil, nil
}
