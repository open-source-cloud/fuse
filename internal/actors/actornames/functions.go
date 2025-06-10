package actornames

import (
	"fmt"
	"github.com/open-source-cloud/fuse/internal/workflow"
)

// WorkflowHandlerName helper function to generate the WorkflowHandler actor name in the context of a workflow instance (from a workflow.ID)
func WorkflowHandlerName(workflowID workflow.ID) string {
	return WorkflowHandlerNameFromStr(workflowID.String())
}

// WorkflowHandlerNameFromStr helper function to generate the WorkflowHandler actor name in the context of a workflow instance (from a string)
func WorkflowHandlerNameFromStr(workflowID string) string {
	return fmt.Sprintf("workflow_handler_%s", workflowID)
}
