// Package concurrency provides bounded concurrency control primitives
package concurrency

import "sync"

// Semaphore provides bounded concurrency control with FIFO queueing
type Semaphore struct {
	mu     sync.Mutex
	limit  int
	active int
	queue  []chan struct{}
}

// NewSemaphore creates a new semaphore with the given concurrency limit
func NewSemaphore(limit int) *Semaphore {
	return &Semaphore{limit: limit, queue: make([]chan struct{}, 0)}
}

// Acquire blocks until a slot is available. Returns a release function.
func (s *Semaphore) Acquire() func() {
	s.mu.Lock()
	if s.active < s.limit {
		s.active++
		s.mu.Unlock()
		return s.release
	}
	// Queue the request (FIFO)
	ch := make(chan struct{})
	s.queue = append(s.queue, ch)
	s.mu.Unlock()
	<-ch // Block until released
	return s.release
}

// TryAcquire returns a release function and true if a slot is immediately available.
// Returns nil, false if no slot is available.
func (s *Semaphore) TryAcquire() (func(), bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active < s.limit {
		s.active++
		return s.release, true
	}
	return nil, false
}

func (s *Semaphore) release() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.queue) > 0 {
		// Wake up the next queued request
		ch := s.queue[0]
		s.queue = s.queue[1:]
		close(ch)
	} else {
		s.active--
	}
}

// Active returns the current number of active acquisitions
func (s *Semaphore) Active() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.active
}

// Queued returns the number of waiting acquisitions
func (s *Semaphore) Queued() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.queue)
}
