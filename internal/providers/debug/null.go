package debug

import (
	"fmt"
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/rs/zerolog/log"
)

type nullNode struct{}

func (n *nullNode) ID() string {
	return fmt.Sprintf("%s/null", debugProviderID)
}

func (n *nullNode) Params() workflow.Params {
	return workflow.Params{}
}

func (n *nullNode) Execute(input workflow.NodeInput) (workflow.NodeResult, error) {
	log.Info().Msgf("NullNode executed with input: %s", input)
	return workflow.NewNodeResult(workflow.NodeOutputStatusSuccess, nil), nil
}
