package workflow

import (
	"github.com/open-source-cloud/fuse/pkg/store"
	"sync"
)

const (
	ThreadRunning  State = "running"
	ThreadWaiting  State = "waiting"
	ThreadFinished State = "finished"
)

func newThreads() *threads {
	return &threads{
		threads: make(map[int]*thread),
		mu:      &sync.Mutex{},
	}
}

func newThread(id int, execID string) *thread {
	return newThreadWithParents(id, execID, []int{})
}

func newThreadWithParents(id int, execID string, parentThreads []int) *thread {
	return &thread{
		id:               id,
		parentThreads:    parentThreads,
		currentExecID:    execID,
		aggregatedOutput: store.New(),
		state:            ThreadRunning,
	}
}

type (
	threads struct {
		threads map[int]*thread
		mu      *sync.Mutex
	}

	thread struct {
		id            int
		parentThreads []int
		currentExecID    string
		aggregatedOutput *store.KV
		state            State
	}
)

func (t *threads) Get(threadID int) *thread {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.threads[threadID]
}

func (t *threads) New(threadID int, execID string) *thread {
	t.mu.Lock()
	defer t.mu.Unlock()
	createdThread := newThread(threadID, execID)
	t.threads[threadID] = createdThread
	return createdThread
}

func (t *threads) NewChild(threadID int, execID string, parentThreads []int) *thread {
	t.mu.Lock()
	defer t.mu.Unlock()
	createdThread := newThreadWithParents(threadID, execID, parentThreads)
	t.threads[threadID] = createdThread
	return createdThread
}

func (t *thread) ID() int {
	return t.id
}

func (t* thread) CurrentExecID() string {
	return t.currentExecID
}

func (t *thread) SetCurrentExecID(execID string, clearStore bool) {
	t.currentExecID = execID
	if clearStore {
		t.aggregatedOutput = store.New()
	}
}

func (t *thread) State() State {
	return t.state
}

func (t *thread) SetState(state State) {
	t.state = state
}

func (t *thread) AggregatedOutput() *store.KV {
	return t.aggregatedOutput
}
