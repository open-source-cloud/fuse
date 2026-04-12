// Package readiness provides a flag to signal application startup completion.
package readiness

import "sync/atomic"

// Flag signals whether the Fuse application has fully started.
// It is set to true in Fuse.Start() after all actors are running.
type Flag struct {
	ready atomic.Bool
}

// NewFlag creates a new Flag (initially false).
func NewFlag() *Flag {
	return &Flag{}
}

// SetReady marks the application as ready.
func (f *Flag) SetReady() {
	f.ready.Store(true)
}

// IsReady returns true if the application has fully started.
func (f *Flag) IsReady() bool {
	return f.ready.Load()
}

// IsNotReady returns true if the application has not fully started.
func (f *Flag) IsNotReady() bool {
	return !f.IsReady()
}
