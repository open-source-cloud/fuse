// Package workflow public interfaces, structs and functions for Workflow
package workflow

import (
	"encoding/json"

	"github.com/open-source-cloud/fuse/pkg/store"
)

// FunctionInput node input type
type FunctionInput struct {
	store *store.KV
}

// NewFunctionInputWith creates a new FunctionInput initialized with provided data
func NewFunctionInputWith(data map[string]any) (*FunctionInput, error) {
	st, err := store.NewWith(data)
	if err != nil {
		return nil, err
	}
	return &FunctionInput{
		store: st,
	}, nil
}

// Get returns the value for the given key
func (i *FunctionInput) Get(key string) any {
	return i.store.Get(key)
}

// GetStr returns the value for the given key as a string
func (i *FunctionInput) GetStr(key string) string {
	return i.store.GetStr(key)
}

// GetInt returns the value for a given key as an int
func (i *FunctionInput) GetInt(key string) int {
	return i.store.GetInt(key)
}

// GetIntSlice returns the value for the given key as an int slice
func (i *FunctionInput) GetIntSlice(key string) []int {
	return i.store.GetIntSlice(key)
}

// GetIntSliceOrDefault returns the value of a given key as an int slice or the default value if nil
func (i *FunctionInput) GetIntSliceOrDefault(key string, defaultValue []int) []int {
	value := i.store.GetIntSlice(key)

	if value == nil {
		return defaultValue
	}
	return value
}

// GetMap returns the value for the given key as a map[string]string
func (i *FunctionInput) GetMap(key string) map[string]string {
	emptyMap := make(map[string]string)

	value := i.store.Get(key)
	if value == nil {
		return emptyMap
	}

	if isString, ok := value.(string); ok {
		var result map[string]string
		err := json.Unmarshal([]byte(isString), &result)
		if err != nil {
			return emptyMap
		}
		return result
	}

	if tryValue, ok := value.(map[string]string); ok {
		return tryValue
	}

	return emptyMap
}

// GetFloat64SliceOrDefault returns the value of a given key as a float64 slice or the default value if nil
func (i *FunctionInput) GetFloat64SliceOrDefault(key string, defaultValue []float64) []float64 {
	value := i.store.GetFloat64Slice(key)

	if value == nil {
		return defaultValue
	}
	return value
}

// GetAnySliceOrDefault returns the value for the given key as an any slice or default value if nil
func (i *FunctionInput) GetAnySliceOrDefault(key string, defaultValue []any) []any {
	value := i.store.Get(key)

	if value == nil {
		return defaultValue
	}
	if tryValue, ok := value.([]any); ok {
		return tryValue
	}

	return defaultValue
}

// Set sets the value for the given key
func (i *FunctionInput) Set(key string, value any) {
	i.store.Set(key, value)
}

// Raw returns the underlying map of all key-value pairs stored in the FunctionInput.
func (i *FunctionInput) Raw() map[string]any {
	return i.store.Raw()
}
