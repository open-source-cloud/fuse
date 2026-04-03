package actors

import (
	"testing"

	"ergo.services/ergo/gen"
	"github.com/stretchr/testify/assert"
)

const (
	testTimerExec1 = "exec-1"
	testTimerExec2 = "exec-2"
	testTimerExec3 = "exec-3"
)

func TestExecutionTimer_CancelAll_EmptyTimers(t *testing.T) {
	timer := NewExecutionTimer()

	// Should not panic on empty map
	timer.CancelAll()

	assert.NotNil(t, timer)
}

func TestExecutionTimer_CancelAll_WithTimers(t *testing.T) {
	timer := NewExecutionTimer()

	cancelled := make(map[string]bool)
	mockCancel := func(id string) gen.CancelFunc {
		return func() bool {
			cancelled[id] = true
			return true
		}
	}

	// Simulate adding timers manually (bypassing Start which needs a process)
	timer.mu.Lock()
	timer.timers[testTimerExec1] = mockCancel(testTimerExec1)
	timer.timers[testTimerExec2] = mockCancel(testTimerExec2)
	timer.timers[testTimerExec3] = mockCancel(testTimerExec3)
	timer.mu.Unlock()

	timer.CancelAll()

	assert.True(t, cancelled[testTimerExec1])
	assert.True(t, cancelled[testTimerExec2])
	assert.True(t, cancelled[testTimerExec3])
	assert.Empty(t, timer.timers)
}

func TestExecutionTimer_Cancel_NonExistent(_ *testing.T) {
	timer := NewExecutionTimer()

	// Should not panic on non-existent key
	timer.Cancel("does-not-exist")
}
