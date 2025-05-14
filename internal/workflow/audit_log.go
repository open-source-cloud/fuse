package workflow

import (
	"github.com/open-source-cloud/fuse/pkg/workflow"
)

type auditLogEntry struct {
	Input  map[string]any           `json:"input,omitempty"`
	Result *workflow.FunctionResult `json:"result,omitempty"`
}

func newAuditLogEntry() *auditLogEntry {
	return &auditLogEntry{
		Input:  nil,
		Result: nil,
	}
}
