package actors

import (
	"sync"
	"time"

	"ergo.services/ergo/gen"
	"github.com/open-source-cloud/fuse/internal/messaging"
)

// ExecutionTimer manages timeout timers for in-flight function executions
type ExecutionTimer struct {
	mu     sync.Mutex
	timers map[string]gen.CancelFunc // execID -> cancel function
}

// NewExecutionTimer creates a new ExecutionTimer
func NewExecutionTimer() *ExecutionTimer {
	return &ExecutionTimer{timers: make(map[string]gen.CancelFunc)}
}

// Start begins a timeout countdown for an execution.
// When the timeout fires, it sends a TimeoutMessage to the handler.
func (et *ExecutionTimer) Start(process interface {
	SendAfter(to any, message any, after time.Duration) (gen.CancelFunc, error)
}, target any, execID string, timeout time.Duration) {
	et.mu.Lock()
	defer et.mu.Unlock()
	cancel, err := process.SendAfter(target, messaging.NewTimeoutMessage(execID), timeout)
	if err != nil {
		return
	}
	et.timers[execID] = cancel
}

// Cancel stops a pending timeout (called when the function completes in time)
func (et *ExecutionTimer) Cancel(execID string) {
	et.mu.Lock()
	defer et.mu.Unlock()
	if cancel, exists := et.timers[execID]; exists {
		cancel()
		delete(et.timers, execID)
	}
}

// CancelAll stops all pending timeouts (called on workflow cancellation)
func (et *ExecutionTimer) CancelAll() {
	et.mu.Lock()
	defer et.mu.Unlock()
	for execID, cancel := range et.timers {
		cancel()
		delete(et.timers, execID)
	}
}
