package actors

import (
	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"fmt"
	"github.com/open-source-cloud/fuse/internal/workflow"
)

type WorkflowFuncPoolFactory Factory[*WorkflowFuncPool]

func NewWorkflowFuncPoolFactory(workflowFunc *WorkflowFuncFactory) *WorkflowFuncPoolFactory {
	return &WorkflowFuncPoolFactory{
		Factory: func() gen.ProcessBehavior {
			return &WorkflowFuncPool{
				workflowFunc: workflowFunc,
			}
		},
	}
}

func WorkflowFuncPoolName(workflowID workflow.ID) string {
	return fmt.Sprintf("workflow_func_pool_%s", workflowID.String())
}

type WorkflowFuncPool struct {
	act.Pool

	workflowFunc *WorkflowFuncFactory
}

func (a *WorkflowFuncPool) Init(_ ...any) (act.PoolOptions, error) {
	opts := act.PoolOptions{
		WorkerFactory: a.workflowFunc.Factory,
		PoolSize:      3,
	}

	a.Log().Info("starting pool %s with %d workers", a.PID(), opts.PoolSize)
	return opts, nil
}
