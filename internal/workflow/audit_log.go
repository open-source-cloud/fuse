package workflow

import (
	"github.com/open-source-cloud/fuse/pkg/workflow"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

// NewAuditLog creates a new AuditLog
func NewAuditLog() *AuditLog {
	return &AuditLog{
		log: orderedmap.New[string, *AuditLogEntry](),
	}
}

// NewAuditLogEntry creates a new AuditLog entry
func NewAuditLogEntry(threadID int, functionNodeID string, input map[string]any) *AuditLogEntry {
	return &AuditLogEntry{
		ThreadID:       threadID,
		FunctionNodeID: functionNodeID,
		Input:          input,
		Result:         nil,
	}
}

type (
	// AuditLog audit log for workflows with an ordered map
	AuditLog struct {
		log *orderedmap.OrderedMap[string, *AuditLogEntry]
	}

	// AuditLogEntry an audit log entry
	AuditLogEntry struct {
		ThreadID       int                      `json:"thread_id"`
		FunctionNodeID string                   `json:"function_node_id"`
		Input          map[string]any           `json:"input,omitempty"`
		Result         *workflow.FunctionResult `json:"result,omitempty"`
	}
)

// NewEntry creates a new entry within the context of an AuditLog instance
func (a *AuditLog) NewEntry(threadID int, functionNodeID string, functionExecID string, input map[string]any) *AuditLogEntry {
	newEntry := NewAuditLogEntry(threadID, functionNodeID, input)
	a.log.Set(functionExecID, newEntry)
	return newEntry
}

// Get gets an AuditLog entry based on the function execution ID
func (a *AuditLog) Get(functionExecID string) (*AuditLogEntry, bool) {
	entry, exists := a.log.Get(functionExecID)
	if !exists {
		return nil, false
	}
	return entry, true
}

// MarshalJSON transforms an AuditLog into JSON
func (a *AuditLog) MarshalJSON() ([]byte, error) {
	return a.log.MarshalJSON()
}
