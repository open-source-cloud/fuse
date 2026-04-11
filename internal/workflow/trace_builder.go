package workflow

import (
	"fmt"
	"time"
)

// BuildTrace constructs an ExecutionTrace from journal entries.
// This is a pure projection — no side effects.
func BuildTrace(workflowID, schemaID string, entries []JournalEntry) *ExecutionTrace {
	trace := &ExecutionTrace{
		WorkflowID: workflowID,
		SchemaID:   schemaID,
		Steps:      make([]ExecutionStepTrace, 0),
	}

	// Track steps by execID for updating across multiple journal entries
	stepIdx := make(map[string]int) // execID -> index in trace.Steps

	for _, entry := range entries {
		switch entry.Type {
		case JournalStepStarted:
			step := ExecutionStepTrace{
				ExecID:         entry.ExecID,
				ThreadID:       entry.ThreadID,
				FunctionNodeID: entry.FunctionNodeID,
				StartedAt:      entry.Timestamp,
				Input:          entry.Input,
				Status:         "running",
				Attempt:        1,
			}
			stepIdx[entry.ExecID] = len(trace.Steps)
			trace.Steps = append(trace.Steps, step)

		case JournalStepCompleted:
			if idx, ok := stepIdx[entry.ExecID]; ok {
				ts := entry.Timestamp
				dur := ts.Sub(trace.Steps[idx].StartedAt)
				trace.Steps[idx].CompletedAt = &ts
				trace.Steps[idx].Duration = durationStr(dur)
				trace.Steps[idx].Status = "completed"
				if entry.Result != nil {
					trace.Steps[idx].Output = &entry.Result.Output
				}
			}

		case JournalStepFailed:
			if idx, ok := stepIdx[entry.ExecID]; ok {
				ts := entry.Timestamp
				dur := ts.Sub(trace.Steps[idx].StartedAt)
				trace.Steps[idx].CompletedAt = &ts
				trace.Steps[idx].Duration = durationStr(dur)
				trace.Steps[idx].Status = "failed"
				if entry.Result != nil && entry.Result.Output.Data != nil {
					if errMsg, exists := entry.Result.Output.Data["error"]; exists {
						s := fmt.Sprintf("%v", errMsg)
						trace.Steps[idx].Error = &s
					}
				}
			}

		case JournalStepRetrying:
			if idx, ok := stepIdx[entry.ExecID]; ok {
				trace.Steps[idx].Status = "retrying"
				trace.Steps[idx].Attempt++
			}

		case JournalStateChanged:
			trace.Status = entry.State
			if entry.State == StateRunning && trace.TriggeredAt.IsZero() {
				trace.TriggeredAt = entry.Timestamp
			}
			if isTerminal(entry.State) {
				ts := entry.Timestamp
				trace.CompletedAt = &ts
				if !trace.TriggeredAt.IsZero() {
					dur := ts.Sub(trace.TriggeredAt)
					trace.Duration = durationStr(dur)
				}
			}
			if entry.State == StateError {
				// Try to extract error from the last failed step
				for i := len(trace.Steps) - 1; i >= 0; i-- {
					if trace.Steps[i].Error != nil {
						trace.Error = trace.Steps[i].Error
						break
					}
				}
			}
		}
	}

	return trace
}

func durationStr(d time.Duration) *string {
	s := d.String()
	return &s
}
