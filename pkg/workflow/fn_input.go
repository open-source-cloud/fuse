package workflow

import "github.com/open-source-cloud/fuse/pkg/store"

// FunctionInput node input type
type FunctionInput struct {
	store *store.KV
}

// NewFunctionInput creates a new FunctionInput object with the given data
func NewFunctionInput() *FunctionInput {
	return &FunctionInput{
		store: store.New(),
	}
}

func NewFunctionInputWith(data map[string]any) (*FunctionInput, error) {
	st, err := store.Init(data)
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

func (i *FunctionInput) GetInt(key string) int {
	return i.store.GetInt(key)
}

// GetIntSlice returns the value for the given key as an int slice
func (i *FunctionInput) GetIntSlice(key string) []int {
	return i.store.GetIntSlice(key)
}

// GetIntSliceOrDefault returns the value of a given key as an int slice or the default value if nil
func (i *FunctionInput) GetIntSliceOrDefault(key string, defaultValue []int) []int {
	value := i.store.Get(key)

	if value == nil {
		return defaultValue
	}
	if tryValue, ok := value.([]int); ok {
		return tryValue
	}

	return defaultValue
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
