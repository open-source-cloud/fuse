package workflow

// BuildExecutionSnapshot constructs an ExecutionSnapshot from journal entries
// and aggregated outputs. This is a pure projection — no side effects.
func BuildExecutionSnapshot(
	workflowID string,
	schemaID string,
	state State,
	entries []JournalEntry,
	aggregatedOutputs map[string]any,
) *ExecutionSnapshot {
	snap := &ExecutionSnapshot{
		SchemaVersion:     1,
		WorkflowID:        workflowID,
		SchemaID:          schemaID,
		Status:            state.String(),
		AggregatedOutputs: aggregatedOutputs,
		Threads:           make([]SnapshotThread, 0),
		NodeRuns:          make([]SnapshotNodeRun, 0),
		Timeline:          make([]SnapshotTimelineEvent, 0, len(entries)),
	}

	// Track node runs by execID for updating across multiple journal entries
	nodeRunIdx := make(map[string]int) // execID -> index in snap.NodeRuns
	// Track threads by threadID
	threadMap := make(map[uint16]*SnapshotThread)
	// Track retry counts per nodeID
	retryCount := make(map[string]int) // nodeID -> retry count

	for _, e := range entries {
		// Build timeline for every entry
		snap.Timeline = append(snap.Timeline, SnapshotTimelineEvent{
			Sequence:  e.Sequence,
			Timestamp: e.Timestamp,
			Type:      string(e.Type),
			ThreadID:  e.ThreadID,
			ExecID:    e.ExecID,
			NodeID:    e.FunctionNodeID,
		})

		switch e.Type {
		case JournalThreadCreated:
			th := &SnapshotThread{
				ThreadID:      e.ThreadID,
				ParentThreads: e.ParentThreads,
				State:         "running",
			}
			threadMap[e.ThreadID] = th

		case JournalThreadDone:
			if th, ok := threadMap[e.ThreadID]; ok {
				th.State = "finished"
			}

		case JournalStepStarted:
			ts := e.Timestamp
			run := SnapshotNodeRun{
				ExecID:               e.ExecID,
				NodeID:               e.FunctionNodeID,
				ThreadID:             e.ThreadID,
				Input:                e.Input,
				Status:               "started",
				StartedAt:            &ts,
				JournalSequenceStart: e.Sequence,
			}
			nodeRunIdx[e.ExecID] = len(snap.NodeRuns)
			snap.NodeRuns = append(snap.NodeRuns, run)

			// Update thread's last exec
			if th, ok := threadMap[e.ThreadID]; ok {
				th.LastExecID = e.ExecID
			}

			// Set workflow startedAt from the first step
			if snap.StartedAt == nil {
				snap.StartedAt = &ts
			}

		case JournalStepCompleted:
			if idx, ok := nodeRunIdx[e.ExecID]; ok {
				ts := e.Timestamp
				snap.NodeRuns[idx].Status = "completed"
				snap.NodeRuns[idx].FinishedAt = &ts
				snap.NodeRuns[idx].JournalSequenceEnd = e.Sequence
				if e.Result != nil {
					snap.NodeRuns[idx].Output = e.Result.Output.Data
				}
			}

		case JournalStepFailed:
			if idx, ok := nodeRunIdx[e.ExecID]; ok {
				ts := e.Timestamp
				snap.NodeRuns[idx].Status = "failed"
				snap.NodeRuns[idx].FinishedAt = &ts
				snap.NodeRuns[idx].JournalSequenceEnd = e.Sequence
				if e.Result != nil && e.Result.Output.Data != nil {
					if errMsg, ok := e.Result.Output.Data["error"]; ok {
						s := errMsg.(string)
						snap.Error = &s
					}
				}
			}

		case JournalStepRetrying:
			if idx, ok := nodeRunIdx[e.ExecID]; ok {
				nodeID := snap.NodeRuns[idx].NodeID
				retryCount[nodeID]++
				snap.NodeRuns[idx].Status = "retrying"
				snap.NodeRuns[idx].RetryAttempt = retryCount[nodeID]
			}

		case JournalStateChanged:
			snap.Status = e.State.String()
			if isTerminal(e.State) {
				ts := e.Timestamp
				snap.FinishedAt = &ts
			}
		}
	}

	// Collect threads into slice
	for _, th := range threadMap {
		snap.Threads = append(snap.Threads, *th)
	}

	return snap
}

func isTerminal(s State) bool {
	return s == StateFinished || s == StateError || s == StateCancelled
}
