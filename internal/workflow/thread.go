package workflow

import (
	"sync"

	"github.com/open-source-cloud/fuse/pkg/workflow"
)

const (
	// ThreadRunning thread running state
	ThreadRunning State = "running"
	// ThreadFinished thread finished state
	ThreadFinished State = "finished"
)

type (
	threads struct {
		threads map[uint16]*thread
		mu      *sync.RWMutex
		// maxID is the highest thread ID assigned so far (static or dynamic).
		// Used to seed AllocateDynamicID so it never collides with static threads.
		maxID uint16
	}

	thread struct {
		id            uint16
		currentExecID workflow.ExecID
		state         State
	}
)

func newThreads() *threads {
	return &threads{
		threads: make(map[uint16]*thread),
		mu:      &sync.RWMutex{},
	}
}

func newThread(id uint16, execID workflow.ExecID) *thread {
	return &thread{
		id:            id,
		currentExecID: execID,
		state:         ThreadRunning,
	}
}

func (t *threads) New(threadID uint16, execID workflow.ExecID) *thread {
	t.mu.Lock()
	defer t.mu.Unlock()
	createdThread := newThread(threadID, execID)
	t.threads[threadID] = createdThread
	if threadID > t.maxID {
		t.maxID = threadID
	}
	return createdThread
}

// AllocateDynamicID allocates a fresh thread ID that does not conflict with any
// existing (static or dynamic) thread in the map. It should be called when
// spawning a new execution thread at runtime (e.g. a ForEach iteration thread).
func (t *threads) AllocateDynamicID() uint16 {
	t.mu.Lock()
	defer t.mu.Unlock()
	candidate := t.maxID
	for {
		candidate++
		if _, exists := t.threads[candidate]; !exists {
			t.maxID = candidate
			return candidate
		}
	}
}

func (t *threads) Get(threadID uint16) *thread {
	t.mu.RLock()
	defer t.mu.RUnlock()
	thread, exists := t.threads[threadID]
	if !exists {
		return nil
	}
	return thread
}

// AllFinished returns true if all threads have reached the ThreadFinished state.
// Returns true for an empty thread map (no threads = nothing pending).
func (t *threads) AllFinished() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	for _, th := range t.threads {
		if th.state != ThreadFinished {
			return false
		}
	}
	return true
}

func (t *threads) AreAllParentsFinishedFor(parentThreadIDs []uint16) bool {
	for _, parentThreadID := range parentThreadIDs {
		parentThread := t.Get(parentThreadID)
		if parentThread == nil || parentThread.State() != ThreadFinished {
			return false
		}
	}
	return true
}

func (t *thread) ID() uint16 {
	return t.id
}

func (t *thread) CurrentExecID() workflow.ExecID {
	return t.currentExecID
}

func (t *thread) SetCurrentExecID(execID workflow.ExecID) {
	t.currentExecID = execID
}

func (t *thread) State() State {
	return t.state
}

func (t *thread) SetState(state State) {
	t.state = state
}
