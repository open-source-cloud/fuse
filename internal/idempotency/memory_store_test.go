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

func TestMemoryStore_CheckAndSet_NewKey(t *testing.T) {
	store := newTestStore(t)

	existingID, existed := store.CheckAndSet("cas-key", "wf-new", 1*time.Hour)
	assert.False(t, existed)
	assert.Empty(t, existingID)

	// Verify it was set
	wfID, exists := store.Check("cas-key")
	assert.True(t, exists)
	assert.Equal(t, "wf-new", wfID)
}

func TestMemoryStore_CheckAndSet_ExistingKey(t *testing.T) {
	store := newTestStore(t)

	// First set
	_, _ = store.CheckAndSet("cas-key", "wf-first", 1*time.Hour)

	// Second attempt with different workflow ID
	existingID, existed := store.CheckAndSet("cas-key", "wf-second", 1*time.Hour)
	assert.True(t, existed)
	assert.Equal(t, "wf-first", existingID)

	// Original value unchanged
	wfID, _ := store.Check("cas-key")
	assert.Equal(t, "wf-first", wfID)
}

func TestMemoryStore_CheckAndSet_ExpiredKey(t *testing.T) {
	store := newTestStore(t)

	_ = store.Set("cas-key", "wf-old", 1*time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	// Expired key should be treated as new
	existingID, existed := store.CheckAndSet("cas-key", "wf-new", 1*time.Hour)
	assert.False(t, existed)
	assert.Empty(t, existingID)
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
