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
	//TODO implement me
	panic("implement me")
}

func (n *nullNode) Execute() (interface{}, error) {
	log.Info().Msg("null node executed")
	return nil, nil
}
