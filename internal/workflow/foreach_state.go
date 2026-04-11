package workflow

import (
	"sync"

	"github.com/open-source-cloud/fuse/pkg/workflow"
)

// ForEachState tracks the progress of an in-flight ForEach iteration loop.
//
// Thread safety: all mutating methods are guarded by an internal mutex and are
// safe to call from the WorkflowHandler goroutine while iteration threads send
// results concurrently.
type ForEachState struct {
	mu sync.Mutex

	// ExecID is the exec-ID of the system/foreach function call (the parent node).
	ExecID workflow.ExecID
	// NodeID is the graph node ID of the foreach node (used for edge lookup).
	NodeID string
	// ThreadID is the workflow thread that owns the foreach node.
	ThreadID uint16

	// Items is the full slice of items to iterate over.
	Items []any
	// BatchSize is the number of items per iteration batch.
	BatchSize int
	// Concurrency is the maximum number of batches that may run simultaneously.
	Concurrency int
	// TotalBatches is len(Items) / BatchSize (ceiling division).
	TotalBatches int

	// nextToStart is the index of the next batch that has not yet been spawned.
	nextToStart int
	// Completed is the number of batches that have finished executing.
	Completed int
	// Results holds the output of each batch, indexed by batch number.
	Results []any

	// iterationThreads maps a live iteration thread ID to its batch index.
	iterationThreads map[uint16]int
}

// NewForEachState creates a new ForEachState for the given items and configuration.
func NewForEachState(
	execID workflow.ExecID,
	threadID uint16,
	nodeID string,
	items []any,
	batchSize int,
	concurrency int,
) *ForEachState {
	if batchSize < 1 {
		batchSize = 1
	}
	if concurrency < 1 {
		concurrency = 1
	}
	totalBatches := (len(items) + batchSize - 1) / batchSize
	return &ForEachState{
		ExecID:           execID,
		NodeID:           nodeID,
		ThreadID:         threadID,
		Items:            items,
		BatchSize:        batchSize,
		Concurrency:      concurrency,
		TotalBatches:     totalBatches,
		nextToStart:      0,
		Completed:        0,
		Results:          make([]any, totalBatches),
		iterationThreads: make(map[uint16]int),
	}
}

// GetBatch returns the slice of items belonging to the given batch index.
// It handles the last batch being smaller than BatchSize.
func (s *ForEachState) GetBatch(batchIndex int) []any {
	start := batchIndex * s.BatchSize
	end := start + s.BatchSize
	if end > len(s.Items) {
		end = len(s.Items)
	}
	return s.Items[start:end]
}

// StartBatch records that the given iteration thread has started processing the
// specified batch.  Call this before sending the RunFunctionAction to the pool.
func (s *ForEachState) StartBatch(threadID uint16, batchIndex int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.iterationThreads[threadID] = batchIndex
	if batchIndex >= s.nextToStart {
		s.nextToStart = batchIndex + 1
	}
}

// RecordCompletion marks the iteration thread as finished and stores its result.
//
// Returns:
//   - nextBatch: the index of the next batch to start, or -1 if no more batches
//     are waiting to be started (either all are running or all are done).
//   - allDone: true when every batch has completed.
func (s *ForEachState) RecordCompletion(threadID uint16, result any) (nextBatch int, allDone bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	batchIndex, ok := s.iterationThreads[threadID]
	if !ok {
		return -1, s.Completed >= s.TotalBatches
	}

	s.Results[batchIndex] = result
	s.Completed++
	delete(s.iterationThreads, threadID)

	if s.Completed >= s.TotalBatches {
		return -1, true
	}

	if s.nextToStart < s.TotalBatches {
		nb := s.nextToStart
		s.nextToStart++
		return nb, false
	}

	return -1, false
}

// IsIterationThread returns true if the given thread ID belongs to this
// ForEachState's active iteration threads.
func (s *ForEachState) IsIterationThread(threadID uint16) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.iterationThreads[threadID]
	return ok
}

// InitialBatchCount returns the number of batches that should be started
// concurrently at the beginning of iteration.
func (s *ForEachState) InitialBatchCount() int {
	return min(s.Concurrency, s.TotalBatches)
}
