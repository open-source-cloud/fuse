package concurrency

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSemaphore_AcquireRelease(t *testing.T) {
	sem := NewSemaphore(2)

	release1 := sem.Acquire()
	assert.Equal(t, 1, sem.Active())

	release2 := sem.Acquire()
	assert.Equal(t, 2, sem.Active())

	release1()
	assert.Equal(t, 1, sem.Active())

	release2()
	assert.Equal(t, 0, sem.Active())
}

func TestSemaphore_BlocksWhenFull(t *testing.T) {
	sem := NewSemaphore(1)

	release := sem.Acquire()
	assert.Equal(t, 1, sem.Active())

	acquired := make(chan struct{})
	go func() {
		r := sem.Acquire()
		close(acquired)
		r()
	}()

	// Should be queued, not acquired
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 1, sem.Queued())

	release()

	// Should now acquire
	select {
	case <-acquired:
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for acquire")
	}
}

func TestSemaphore_FIFOOrdering(t *testing.T) {
	sem := NewSemaphore(1)
	release := sem.Acquire()

	order := make([]int, 0, 3)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := range 3 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			r := sem.Acquire()
			mu.Lock()
			order = append(order, idx)
			mu.Unlock()
			r()
		}(i)
		time.Sleep(5 * time.Millisecond) // Ensure ordering in queue
	}

	release()
	wg.Wait()

	assert.Equal(t, []int{0, 1, 2}, order)
}

func TestSemaphore_TryAcquire(t *testing.T) {
	sem := NewSemaphore(1)

	release, ok := sem.TryAcquire()
	assert.True(t, ok)
	assert.NotNil(t, release)
	assert.Equal(t, 1, sem.Active())

	_, ok = sem.TryAcquire()
	assert.False(t, ok)

	release()
	assert.Equal(t, 0, sem.Active())
}

func TestSemaphore_ConcurrentStress(t *testing.T) {
	sem := NewSemaphore(5)
	var maxConcurrent atomic.Int32
	var current atomic.Int32

	var wg sync.WaitGroup
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			release := sem.Acquire()
			c := current.Add(1)
			for {
				old := maxConcurrent.Load()
				if c <= old || maxConcurrent.CompareAndSwap(old, c) {
					break
				}
			}
			time.Sleep(1 * time.Millisecond)
			current.Add(-1)
			release()
		}()
	}
	wg.Wait()

	assert.LessOrEqual(t, int(maxConcurrent.Load()), 5)
	assert.Equal(t, 0, sem.Active())
	assert.Equal(t, 0, sem.Queued())
}
