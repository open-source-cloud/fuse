package events

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

const handlerTimeout = 30 * time.Second

type subscriber struct {
	id      SubscriptionID
	handler EventHandler
}

// MemoryBus is an in-memory implementation of EventBus
type MemoryBus struct {
	mu          sync.RWMutex
	subscribers map[string][]subscriber // eventType -> subscribers
	nextID      uint64
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewMemoryBus creates a new in-memory event bus
func NewMemoryBus(ctx context.Context) *MemoryBus {
	ctx, cancel := context.WithCancel(ctx)
	return &MemoryBus{
		subscribers: make(map[string][]subscriber),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Publish emits an event to all matching subscribers.
// Handlers are invoked asynchronously in goroutines.
func (b *MemoryBus) Publish(event Event) error {
	b.mu.RLock()
	subs := make([]subscriber, len(b.subscribers[event.Type]))
	copy(subs, b.subscribers[event.Type])
	b.mu.RUnlock()

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	for _, sub := range subs {
		go b.invokeHandler(sub, event)
	}
	return nil
}

// Subscribe registers a callback for events matching the given type
func (b *MemoryBus) Subscribe(eventType string, handler EventHandler) (SubscriptionID, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.nextID++
	id := SubscriptionID(fmt.Sprintf("sub-%d", b.nextID))
	b.subscribers[eventType] = append(b.subscribers[eventType], subscriber{id: id, handler: handler})
	return id, nil
}

// Unsubscribe removes a subscription
func (b *MemoryBus) Unsubscribe(id SubscriptionID) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	for eventType, subs := range b.subscribers {
		for i, sub := range subs {
			if sub.id == id {
				b.subscribers[eventType] = append(subs[:i], subs[i+1:]...)
				return nil
			}
		}
	}
	return nil
}

// Close stops the event bus
func (b *MemoryBus) Close() error {
	b.cancel()
	return nil
}

func (b *MemoryBus) invokeHandler(sub subscriber, event Event) {
	ctx, cancel := context.WithTimeout(b.ctx, handlerTimeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error().Msgf("event handler %s panicked: %v", sub.id, r)
				done <- fmt.Errorf("handler panicked: %v", r)
			}
		}()
		done <- sub.handler(event)
	}()

	select {
	case err := <-done:
		if err != nil {
			log.Error().Err(err).Msgf("event handler %s returned error for event %s", sub.id, event.Type)
		}
	case <-ctx.Done():
		log.Error().Msgf("event handler %s timed out for event %s", sub.id, event.Type)
	}
}
