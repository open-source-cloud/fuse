package actors

import (
	"testing"

	"ergo.services/ergo/gen"
	"github.com/stretchr/testify/assert"
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
	timer.timers["exec-1"] = mockCancel("exec-1")
	timer.timers["exec-2"] = mockCancel("exec-2")
	timer.timers["exec-3"] = mockCancel("exec-3")
	timer.mu.Unlock()

	timer.CancelAll()

	assert.True(t, cancelled["exec-1"])
	assert.True(t, cancelled["exec-2"])
	assert.True(t, cancelled["exec-3"])
	assert.Empty(t, timer.timers)
}

func TestExecutionTimer_Cancel_NonExistent(_ *testing.T) {
	timer := NewExecutionTimer()

	// Should not panic on non-existent key
	timer.Cancel("does-not-exist")
}
