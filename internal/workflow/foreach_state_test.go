package workflow

import (
	"testing"

	"github.com/open-source-cloud/fuse/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helpers

func newTestForEachState(items []any, batchSize, concurrency int) *ForEachState {
	return NewForEachState(
		workflow.NewExecID(0),
		0,
		"foreach1",
		items,
		batchSize,
		concurrency,
	)
}

// --- NewForEachState ---

func TestNewForEachState_SingleItems(t *testing.T) {
	items := []any{"a", "b", "c"}
	s := newTestForEachState(items, 1, 1)

	assert.Equal(t, 3, s.TotalBatches)
	assert.Equal(t, 1, s.BatchSize)
	assert.Equal(t, 1, s.Concurrency)
	assert.Len(t, s.Results, 3)
}

func TestNewForEachState_BatchedItems(t *testing.T) {
	items := []any{1, 2, 3, 4, 5}
	s := newTestForEachState(items, 2, 1)

	// 5 items in batches of 2 → ceil(5/2) = 3 batches
	assert.Equal(t, 3, s.TotalBatches)
	assert.Equal(t, 2, s.BatchSize)
}

func TestNewForEachState_ExactDivision(t *testing.T) {
	items := []any{1, 2, 3, 4}
	s := newTestForEachState(items, 2, 1)

	assert.Equal(t, 2, s.TotalBatches)
}

func TestNewForEachState_SingleItem(t *testing.T) {
	items := []any{"only"}
	s := newTestForEachState(items, 1, 1)

	assert.Equal(t, 1, s.TotalBatches)
}

func TestNewForEachState_DefaultsOnInvalidConfig(t *testing.T) {
	s := NewForEachState(workflow.NewExecID(0), 0, "n1", []any{"x"}, 0, 0)
	// batchSize and concurrency should be normalised to 1
	assert.Equal(t, 1, s.BatchSize)
	assert.Equal(t, 1, s.Concurrency)
}

// --- GetBatch ---

func TestGetBatch_FirstBatch(t *testing.T) {
	items := []any{"a", "b", "c", "d", "e"}
	s := newTestForEachState(items, 2, 1)

	batch := s.GetBatch(0)
	assert.Equal(t, []any{"a", "b"}, batch)
}

func TestGetBatch_LastBatchSmallerThanBatchSize(t *testing.T) {
	items := []any{"a", "b", "c"}
	s := newTestForEachState(items, 2, 1)

	// batch index 1 → items[2:3] = ["c"]
	batch := s.GetBatch(1)
	assert.Equal(t, []any{"c"}, batch)
}

func TestGetBatch_SingleItemBatch(t *testing.T) {
	items := []any{10, 20, 30}
	s := newTestForEachState(items, 1, 1)

	assert.Equal(t, []any{10}, s.GetBatch(0))
	assert.Equal(t, []any{20}, s.GetBatch(1))
	assert.Equal(t, []any{30}, s.GetBatch(2))
}

// --- StartBatch / RecordCompletion ---

func TestRecordCompletion_SingleItem_Sequential(t *testing.T) {
	items := []any{"a", "b", "c"}
	s := newTestForEachState(items, 1, 1)

	// Start batch 0 on thread 10
	s.StartBatch(10, 0)

	nextBatch, allDone := s.RecordCompletion(10, "result-a")
	assert.False(t, allDone)
	assert.Equal(t, 1, nextBatch) // next sequential batch

	s.StartBatch(11, 1)
	nextBatch, allDone = s.RecordCompletion(11, "result-b")
	assert.False(t, allDone)
	assert.Equal(t, 2, nextBatch)

	s.StartBatch(12, 2)
	nextBatch, allDone = s.RecordCompletion(12, "result-c")
	assert.True(t, allDone)
	assert.Equal(t, -1, nextBatch)

	assert.Equal(t, []any{"result-a", "result-b", "result-c"}, s.Results)
}

func TestRecordCompletion_Concurrent(t *testing.T) {
	items := []any{"a", "b", "c", "d"}
	s := newTestForEachState(items, 1, 2) // concurrency=2

	// Start first two batches
	s.StartBatch(10, 0)
	s.StartBatch(11, 1)

	// Batch 0 finishes → next batch is 2
	nextBatch, allDone := s.RecordCompletion(10, "r0")
	assert.False(t, allDone)
	assert.Equal(t, 2, nextBatch)

	s.StartBatch(12, 2)

	// Batch 1 finishes → next batch is 3
	nextBatch, allDone = s.RecordCompletion(11, "r1")
	assert.False(t, allDone)
	assert.Equal(t, 3, nextBatch)

	s.StartBatch(13, 3)

	// Batch 2 finishes → no new batch to start (3 is already running)
	nextBatch, allDone = s.RecordCompletion(12, "r2")
	assert.False(t, allDone)
	assert.Equal(t, -1, nextBatch)

	// Batch 3 finishes → all done
	nextBatch, allDone = s.RecordCompletion(13, "r3")
	assert.True(t, allDone)
	assert.Equal(t, -1, nextBatch)
}

func TestRecordCompletion_AllStartedConcurrently(t *testing.T) {
	items := []any{1, 2, 3}
	s := newTestForEachState(items, 1, 3) // concurrency = total

	s.StartBatch(10, 0)
	s.StartBatch(11, 1)
	s.StartBatch(12, 2)

	// Each completion should not trigger a new batch
	nb, done := s.RecordCompletion(10, "r0")
	assert.False(t, done)
	assert.Equal(t, -1, nb)

	nb, done = s.RecordCompletion(11, "r1")
	assert.False(t, done)
	assert.Equal(t, -1, nb)

	nb, done = s.RecordCompletion(12, "r2")
	assert.True(t, done)
	assert.Equal(t, -1, nb)
}

func TestRecordCompletion_UnknownThread_IsIdempotent(t *testing.T) {
	s := newTestForEachState([]any{"x"}, 1, 1)
	// unknown thread should not panic
	nb, done := s.RecordCompletion(99, "whatever")
	assert.Equal(t, -1, nb)
	assert.False(t, done)
}

// --- IsIterationThread ---

func TestIsIterationThread_ActiveThread(t *testing.T) {
	s := newTestForEachState([]any{"a"}, 1, 1)
	s.StartBatch(42, 0)
	assert.True(t, s.IsIterationThread(42))
}

func TestIsIterationThread_UnknownThread(t *testing.T) {
	s := newTestForEachState([]any{"a"}, 1, 1)
	assert.False(t, s.IsIterationThread(99))
}

func TestIsIterationThread_FinishedThread(t *testing.T) {
	s := newTestForEachState([]any{"a"}, 1, 1)
	s.StartBatch(5, 0)
	s.RecordCompletion(5, nil) //nolint:errcheck
	assert.False(t, s.IsIterationThread(5))
}

// --- InitialBatchCount ---

func TestInitialBatchCount_CappedByConcurrency(t *testing.T) {
	s := newTestForEachState([]any{1, 2, 3, 4, 5}, 1, 2)
	assert.Equal(t, 2, s.InitialBatchCount())
}

func TestInitialBatchCount_CappedByTotalBatches(t *testing.T) {
	s := newTestForEachState([]any{1, 2}, 1, 10)
	assert.Equal(t, 2, s.InitialBatchCount()) // only 2 batches exist
}

// --- edge cases ---

func TestForEachState_EmptyItems(t *testing.T) {
	// An empty item list should result in 0 batches.
	// (The handler short-circuits before creating a state for empty collections,
	// but the state struct should still handle it gracefully.)
	s := newTestForEachState([]any{}, 1, 1)
	require.Equal(t, 0, s.TotalBatches)
	assert.Equal(t, 0, s.InitialBatchCount())
}

func TestForEachState_LargeBatch(t *testing.T) {
	items := make([]any, 100)
	for i := range items {
		items[i] = i
	}
	s := newTestForEachState(items, 10, 5)

	assert.Equal(t, 10, s.TotalBatches)
	assert.Equal(t, 5, s.InitialBatchCount())

	// First batch should be items 0-9
	batch := s.GetBatch(0)
	assert.Len(t, batch, 10)
	assert.Equal(t, 0, batch[0])
	assert.Equal(t, 9, batch[9])
}
