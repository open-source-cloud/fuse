package objectstore_test

import (
	"testing"

	"github.com/open-source-cloud/fuse/pkg/objectstore"
	"github.com/stretchr/testify/require"
)

func TestFilesystemObjectStore(t *testing.T) {
	basePath := t.TempDir()
	store, err := objectstore.NewFilesystemObjectStore(basePath)
	require.NoError(t, err)

	testObjectStore(t, store)
}
