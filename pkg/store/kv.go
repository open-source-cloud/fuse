// Package store provides a simple key-value store with support for concurrency, dot notation and typed values
package store

import (
	"fmt"
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

// NewWith initializes a new KV store from rawData, converting the input map to a thread-safe key-value store.
// It returns a pointer to the initialized KV store or an error if the initialization fails.
func NewWith(rawData map[string]any) (*KV, error) {
	data := objx.New(rawData)
	return &KV{
		data: data,
	}, nil
}

// Raw returns the underlying map[string]any representing all key-value pairs in the store without any modifications.
func (k *KV) Raw() map[string]any {
	return k.data
}

// MergeWith merges provided data into the current Map
func (k *KV) MergeWith(data map[string]any) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.data = k.data.Merge(data)
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
	return val.Data()
}

// GetStr returns the value of a key as a string
func (k *KV) GetStr(key string) string {
	k.mu.RLock()
	defer k.mu.RUnlock()
	val := k.data.Get(key)
	return val.Str()
}

// GetInt returns the value of a key as an int
func (k *KV) GetInt(key string) int {
	k.mu.RLock()
	defer k.mu.RUnlock()
	val := k.data.Get(key)
	return val.Int()
}

// GetBool returns the value of a key as a bool
func (k *KV) GetBool(key string) bool {
	k.mu.RLock()
	defer k.mu.RUnlock()
	val := k.data.Get(key)
	return val.Bool()
}

// GetFloat returns the value of a key as a float64
func (k *KV) GetFloat(key string) float64 {
	k.mu.RLock()
	defer k.mu.RUnlock()
	val := k.data.Get(key)
	return val.Float64()
}

// GetIntSlice retrieves the value associated with the specified key as a slice of integers.
// Returns nil if the key does not exist or the value is not a slice of integers.
func (k *KV) GetIntSlice(key string) []int {
	k.mu.RLock()
	defer k.mu.RUnlock()
	val := k.data.Get(key)
	return val.IntSlice()
}

// GetFloat64Slice retrieves the value associated with the specified key as a slice of float64 numbers.
// Returns nil if the key does not exist or the value is not a slice of integers.
func (k *KV) GetFloat64Slice(key string) []float64 {
	k.mu.RLock()
	defer k.mu.RUnlock()
	val := k.data.Get(key)
	return val.Float64Slice()
}

// GetMapStr returns the value of a key as a map[string]any
func (k *KV) GetMapStr(key string) map[string]string {
	k.mu.RLock()
	defer k.mu.RUnlock()
	rawVal := k.data.Get(key)
	val := rawVal.Data()
	if val == nil {
		return nil
	}
	if mapVal, ok := val.(map[string]any); ok {
		mapStr := make(map[string]string)
		for k, v := range mapVal {
			mapStr[k] = fmt.Sprintf("%v", v)
		}
		return mapStr
	}
	if mapVal, ok := val.(map[string]string); ok {
		return mapVal
	}
	return nil
}
