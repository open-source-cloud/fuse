package actornames

import (
	"fmt"
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// WorkflowHandlerName helper function to generate the WorkflowHandler actor name in the context of a workflow instance (from a string)
func WorkflowHandlerName(workflowID workflow.ID) string {
	return fmt.Sprintf("workflow_handler_%s", workflowID)
}
