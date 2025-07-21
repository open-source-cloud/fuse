package workflow

import (
	"github.com/open-source-cloud/fuse/pkg/workflow"
	"sync"
)

const (
	// ThreadRunning thread running state
	ThreadRunning State = "running"
	// ThreadFinished thread finished state
	ThreadFinished State = "finished"
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

type (
	threads struct {
		threads map[uint16]*thread
		mu      *sync.RWMutex
	}

	thread struct {
		id            uint16
		currentExecID workflow.ExecID
		state         State
	}
)

func (t *threads) New(threadID uint16, execID workflow.ExecID) *thread {
	t.mu.Lock()
	defer t.mu.Unlock()
	createdThread := newThread(threadID, execID)
	t.threads[threadID] = createdThread
	return createdThread
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

func (t *threads) AreAllParentsFinishedFor(parentThreadIDs []uint16) bool {
	for _, parentThreadID := range parentThreadIDs {
		parentThread := t.Get(parentThreadID)
		if parentThread.State() != ThreadFinished {
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
