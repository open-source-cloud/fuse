package events

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestBus(t *testing.T) *MemoryBus {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	return NewMemoryBus(ctx)
}

func TestMemoryBus_PublishSubscribe(t *testing.T) {
	bus := newTestBus(t)

	received := make(chan Event, 1)
	_, err := bus.Subscribe("test.event", func(e Event) error {
		received <- e
		return nil
	})
	require.NoError(t, err)

	err = bus.Publish(Event{Type: "test.event", Data: map[string]any{"key": "value"}})
	require.NoError(t, err)

	select {
	case e := <-received:
		assert.Equal(t, "test.event", e.Type)
		assert.Equal(t, "value", e.Data["key"])
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestMemoryBus_MultipleSubscribers(t *testing.T) {
	bus := newTestBus(t)

	var count atomic.Int32

	for range 3 {
		_, _ = bus.Subscribe("test.multi", func(_ Event) error {
			count.Add(1)
			return nil
		})
	}

	_ = bus.Publish(Event{Type: "test.multi"})
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, int32(3), count.Load())
}

func TestMemoryBus_Unsubscribe(t *testing.T) {
	bus := newTestBus(t)

	var count atomic.Int32

	id, _ := bus.Subscribe("test.unsub", func(_ Event) error {
		count.Add(1)
		return nil
	})

	_ = bus.Publish(Event{Type: "test.unsub"})
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, int32(1), count.Load())

	err := bus.Unsubscribe(id)
	require.NoError(t, err)

	_ = bus.Publish(Event{Type: "test.unsub"})
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, int32(1), count.Load()) // Should not increase
}

func TestMemoryBus_NoMatchingSubscribers(t *testing.T) {
	bus := newTestBus(t)

	var called atomic.Bool
	_, _ = bus.Subscribe("type-a", func(_ Event) error {
		called.Store(true)
		return nil
	})

	_ = bus.Publish(Event{Type: "type-b"})
	time.Sleep(50 * time.Millisecond)

	assert.False(t, called.Load())
}

func TestMemoryBus_HandlerPanicRecovery(t *testing.T) {
	bus := newTestBus(t)

	var wg sync.WaitGroup
	wg.Add(1)

	_, _ = bus.Subscribe("test.panic", func(_ Event) error {
		defer wg.Done()
		panic("test panic")
	})

	// Should not crash
	_ = bus.Publish(Event{Type: "test.panic"})
	wg.Wait()
}

func TestMemoryBus_ConcurrentPublish(t *testing.T) {
	bus := newTestBus(t)

	var count atomic.Int32
	_, _ = bus.Subscribe("test.concurrent", func(_ Event) error {
		count.Add(1)
		return nil
	})

	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = bus.Publish(Event{Type: "test.concurrent"})
		}()
	}
	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, int32(100), count.Load())
}
