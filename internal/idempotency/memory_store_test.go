package idempotency

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T) *MemoryStore {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	return NewMemoryStore(ctx)
}

func TestMemoryStore_SetAndCheck(t *testing.T) {
	store := newTestStore(t)

	err := store.Set("key-1", "wf-123", 1*time.Hour)
	require.NoError(t, err)

	wfID, exists := store.Check("key-1")
	assert.True(t, exists)
	assert.Equal(t, "wf-123", wfID)
}

func TestMemoryStore_CheckMiss(t *testing.T) {
	store := newTestStore(t)

	wfID, exists := store.Check("nonexistent")
	assert.False(t, exists)
	assert.Empty(t, wfID)
}

func TestMemoryStore_TTLExpiry(t *testing.T) {
	store := newTestStore(t)

	err := store.Set("key-expire", "wf-456", 1*time.Millisecond)
	require.NoError(t, err)

	time.Sleep(5 * time.Millisecond)

	wfID, exists := store.Check("key-expire")
	assert.False(t, exists)
	assert.Empty(t, wfID)
}

func TestMemoryStore_Delete(t *testing.T) {
	store := newTestStore(t)

	err := store.Set("key-del", "wf-789", 1*time.Hour)
	require.NoError(t, err)

	err = store.Delete("key-del")
	require.NoError(t, err)

	wfID, exists := store.Check("key-del")
	assert.False(t, exists)
	assert.Empty(t, wfID)
}

func TestMemoryStore_DeleteNonexistent(t *testing.T) {
	store := newTestStore(t)

	err := store.Delete("nonexistent")
	assert.NoError(t, err)
}

func TestMemoryStore_OverwriteKey(t *testing.T) {
	store := newTestStore(t)

	_ = store.Set("key-1", "wf-old", 1*time.Hour)
	_ = store.Set("key-1", "wf-new", 1*time.Hour)

	wfID, exists := store.Check("key-1")
	assert.True(t, exists)
	assert.Equal(t, "wf-new", wfID)
}

func TestMemoryStore_ConcurrentAccess(t *testing.T) {
	store := newTestStore(t)

	var wg sync.WaitGroup
	for i := range 100 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			key := "key-concurrent"
			wfID := "wf-concurrent"
			_ = store.Set(key, wfID, 1*time.Hour)
			store.Check(key)
			if idx%2 == 0 {
				_ = store.Delete(key)
			}
		}(i)
	}
	wg.Wait()
}
