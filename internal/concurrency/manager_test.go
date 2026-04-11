package concurrency

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestManager_AcquireFunction(t *testing.T) {
	m := NewManager()

	release := m.AcquireFunction("pkg/fn", 2)
	assert.NotNil(t, release)
	release()
}

func TestManager_TryAcquireWorkflow(t *testing.T) {
	m := NewManager()

	release1, ok := m.TryAcquireWorkflow("schema-1", 1)
	assert.True(t, ok)
	assert.NotNil(t, release1)

	_, ok = m.TryAcquireWorkflow("schema-1", 1)
	assert.False(t, ok)

	release1()

	release2, ok := m.TryAcquireWorkflow("schema-1", 1)
	assert.True(t, ok)
	release2()
}

func TestManager_IsolationBetweenFunctions(t *testing.T) {
	m := NewManager()

	release1 := m.AcquireFunction("fn-a", 1)
	release2 := m.AcquireFunction("fn-b", 1)

	// Both should succeed since they are different functions
	assert.NotNil(t, release1)
	assert.NotNil(t, release2)

	release1()
	release2()
}

func TestManager_AcquireKeyed(t *testing.T) {
	m := NewManager()

	release1 := m.AcquireKeyed("fn-a", "user-1", 1)
	release2 := m.AcquireKeyed("fn-a", "user-2", 1)

	// Different keys should not conflict
	assert.NotNil(t, release1)
	assert.NotNil(t, release2)

	release1()
	release2()
}
