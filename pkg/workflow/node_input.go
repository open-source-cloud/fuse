package workflow

import "github.com/open-source-cloud/fuse/pkg/store"

// NodeInput node input type
type NodeInput struct {
	store *store.KV
}

// NewNodeInput creates a new NodeInput object with the given data
func NewNodeInput() *NodeInput {
	return &NodeInput{
		store: store.New(),
	}
}

// Get returns the value for the given key
func (i *NodeInput) Get(key string) any {
	return i.store.Get(key)
}

// GetIntSlice returns the value for the given key as an int slice
func (i *NodeInput) GetIntSlice(key string) []int {
	return i.store.GetIntSlice(key)
}

// Set sets the value for the given key
func (i *NodeInput) Set(key string, value any) {
	i.store.Set(key, value)
}

// Raw returns the underlying map of all key-value pairs stored in the NodeInput.
func (i *NodeInput) Raw() map[string]any {
	return i.store.Raw()
}
