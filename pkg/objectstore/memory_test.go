package objectstore_test

import (
	"testing"

	"github.com/open-source-cloud/fuse/pkg/objectstore"
)

func TestMemoryObjectStore(t *testing.T) {
	store := objectstore.NewMemoryObjectStore()
	testObjectStore(t, store)
}
