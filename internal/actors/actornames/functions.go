package actornames

import (
	"fmt"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// WorkflowHandlerName helper function to generate the WorkflowHandler actor name in the context of a workflow instance
func WorkflowHandlerName(workflowID workflow.ID) string {
	return fmt.Sprintf("workflow_handler_%s", workflowID)
}

// WorkflowInstanceSupervisorName helper function to generate the WorkflowInstanceSupervisor actor name in the context of a workflow instance
func WorkflowInstanceSupervisorName(workflowID workflow.ID) string {
	return fmt.Sprintf("%s_%s", WorkflowInstanceSupervisor, workflowID)
}
