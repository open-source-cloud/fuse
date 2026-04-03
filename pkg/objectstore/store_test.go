package objectstore_test

import (
	"context"
	"testing"

	"github.com/open-source-cloud/fuse/pkg/objectstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testObjectStore runs the full behavioral contract against any ObjectStore implementation.
func testObjectStore(t *testing.T, store objectstore.ObjectStore) {
	t.Helper()
	ctx := context.Background()

	t.Run("Put and Get", func(t *testing.T) {
		// Arrange
		key := "test/put-get.json"
		data := []byte(`{"hello":"world"}`)

		// Act
		err := store.Put(ctx, key, data)
		require.NoError(t, err)

		got, err := store.Get(ctx, key)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, data, got)
	})

	t.Run("Get returns ErrObjectNotFound for missing key", func(t *testing.T) {
		// Act
		_, err := store.Get(ctx, "nonexistent/key.json")

		// Assert
		assert.ErrorIs(t, err, objectstore.ErrObjectNotFound)
	})

	t.Run("Put overwrites existing key", func(t *testing.T) {
		// Arrange
		key := "test/overwrite.json"
		require.NoError(t, store.Put(ctx, key, []byte("first")))

		// Act
		require.NoError(t, store.Put(ctx, key, []byte("second")))
		got, err := store.Get(ctx, key)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, []byte("second"), got)
	})

	t.Run("Delete removes the object", func(t *testing.T) {
		// Arrange
		key := "test/delete.json"
		require.NoError(t, store.Put(ctx, key, []byte("to-delete")))

		// Act
		err := store.Delete(ctx, key)
		require.NoError(t, err)

		// Assert
		_, err = store.Get(ctx, key)
		assert.ErrorIs(t, err, objectstore.ErrObjectNotFound)
	})

	t.Run("Delete is idempotent for missing key", func(t *testing.T) {
		// Act
		err := store.Delete(ctx, "nonexistent/to-delete.json")

		// Assert
		assert.NoError(t, err)
	})

	t.Run("Exists returns true for existing key", func(t *testing.T) {
		// Arrange
		key := "test/exists-true.json"
		require.NoError(t, store.Put(ctx, key, []byte("data")))

		// Act
		exists, err := store.Exists(ctx, key)

		// Assert
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("Exists returns false for missing key", func(t *testing.T) {
		// Act
		exists, err := store.Exists(ctx, "nonexistent/exists-false.json")

		// Assert
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("Put and Get with nested keys", func(t *testing.T) {
		// Arrange
		key := "workflows/abc-123/journal/42/input.json"
		data := []byte(`{"param":"value"}`)

		// Act
		require.NoError(t, store.Put(ctx, key, data))
		got, err := store.Get(ctx, key)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, data, got)
	})

	t.Run("Put and Get empty data", func(t *testing.T) {
		// Arrange
		key := "test/empty.json"

		// Act
		require.NoError(t, store.Put(ctx, key, []byte{}))
		got, err := store.Get(ctx, key)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, []byte{}, got)
	})

	t.Run("Get returns copy not reference", func(t *testing.T) {
		// Arrange
		key := "test/copy.json"
		require.NoError(t, store.Put(ctx, key, []byte("original")))

		// Act
		got1, _ := store.Get(ctx, key)
		got1[0] = 'X' // mutate first copy
		got2, _ := store.Get(ctx, key)

		// Assert - second get should return the original data
		assert.Equal(t, []byte("original"), got2)
	})
}
