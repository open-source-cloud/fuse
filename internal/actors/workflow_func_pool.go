package actors

import (
	"fmt"

	"ergo.services/ergo/act"
	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/workflow"
)

// WorkflowFuncPoolFactory redefines a WorkflowFuncPool actor factory type for better readability
type WorkflowFuncPoolFactory ActorFactory[*WorkflowFuncPool]

// NewWorkflowFuncPoolFactory creates a dependency injection WorkflowFuncPool actor factory
func NewWorkflowFuncPoolFactory(workflowFunc *WorkflowFuncFactory) *WorkflowFuncPoolFactory {
	return &WorkflowFuncPoolFactory{
		Factory: func() gen.ProcessBehavior {
			return &WorkflowFuncPool{
				workflowFunc: workflowFunc,
			}
		},
	}
}

// WorkflowFuncPoolName helper func to generate a WorkflowFuncPool actor name in the context of a workflow instance
func WorkflowFuncPoolName(workflowID workflow.ID) string {
	return fmt.Sprintf("workflow_func_pool_%s", workflowID.String())
}

// WorkflowFuncPool defines a WorkflowFuncPool actor
type WorkflowFuncPool struct {
	act.Pool

	workflowFunc *WorkflowFuncFactory
}

// Init called when a WorkflowFuncPool is getting initialized
func (a *WorkflowFuncPool) Init(_ ...any) (act.PoolOptions, error) {
	opts := act.PoolOptions{
		PoolSize:      3,
		WorkerFactory: a.workflowFunc.Factory,
	}

	a.Log().Debug("starting pool %s with %d workers", a.PID(), opts.PoolSize)
	return opts, nil
}
