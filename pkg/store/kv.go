// Package store provides a simple key-value store with support for concurrency, dot notation and typed values
package store

import (
	"sync"

	"github.com/stretchr/objx"
)

// KV is a simple key-value store with support for concurrency, dot notation and typed values
// It is backed by a map[string]any, which means that it supports any type of value
// It is thread-safe and supports dot notation for nested keys
type KV struct {
	data objx.Map
	mu   sync.RWMutex
}

// New creates a new KV store
func New() *KV {
	rawData := make(map[string]any)
	data := objx.New(rawData)
	return &KV{
		data: data,
	}
}

// Clear clears the store
func (k *KV) Clear() {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.data = objx.New(make(map[string]any))
}

// Has checks if a key exists in the store
func (k *KV) Has(key string) bool {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return k.data.Has(key)
}

// Set sets a key to a value
func (k *KV) Set(key string, value any) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.data.Set(key, value)
}

// Get returns the value of a key
func (k *KV) Get(key string) any {
	k.mu.RLock()
	defer k.mu.RUnlock()
	val := k.data.Get(key)
	if val == nil {
		return nil
	}
	return val.Data()
}

// GetStr returns the value of a key as a string
func (k *KV) GetStr(key string) string {
	k.mu.RLock()
	defer k.mu.RUnlock()
	val := k.data.Get(key)
	if val == nil {
		return ""
	}
	return val.Str()
}

// GetInt returns the value of a key as an int
func (k *KV) GetInt(key string) int {
	k.mu.RLock()
	defer k.mu.RUnlock()
	val := k.data.Get(key)
	if val == nil {
		return 0
	}
	return val.Int()
}

// GetBool returns the value of a key as a bool
func (k *KV) GetBool(key string) bool {
	k.mu.RLock()
	defer k.mu.RUnlock()
	val := k.data.Get(key)
	if val == nil {
		return false
	}
	return val.Bool()
}

// GetFloat returns the value of a key as a float64
func (k *KV) GetFloat(key string) float64 {
	k.mu.RLock()
	defer k.mu.RUnlock()
	val := k.data.Get(key)
	if val == nil {
		return 0
	}
	return val.Float64()
}
