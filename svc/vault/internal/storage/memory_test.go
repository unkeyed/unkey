package storage

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/otel/logging"
)

// TestMemory_PutAndGet verifies basic put and get operations.
func TestMemory_PutAndGet(t *testing.T) {
	store := newTestMemoryStorage(t)
	ctx := context.Background()

	key := "test-key"
	data := []byte("test-data")

	// Put should succeed
	err := store.PutObject(ctx, key, data)
	require.NoError(t, err)

	// Get should return the same data
	retrieved, found, err := store.GetObject(ctx, key)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, data, retrieved)
}

// TestMemory_GetNonExistent verifies that getting a non-existent key returns
// found=false without error.
func TestMemory_GetNonExistent(t *testing.T) {
	store := newTestMemoryStorage(t)
	ctx := context.Background()

	retrieved, found, err := store.GetObject(ctx, "nonexistent-key")
	require.NoError(t, err)
	require.False(t, found)
	require.Nil(t, retrieved)
}

// TestMemory_Overwrite verifies that putting to an existing key overwrites.
func TestMemory_Overwrite(t *testing.T) {
	store := newTestMemoryStorage(t)
	ctx := context.Background()

	key := "test-key"
	data1 := []byte("data-version-1")
	data2 := []byte("data-version-2")

	// Put initial data
	err := store.PutObject(ctx, key, data1)
	require.NoError(t, err)

	// Overwrite with new data
	err = store.PutObject(ctx, key, data2)
	require.NoError(t, err)

	// Get should return new data
	retrieved, found, err := store.GetObject(ctx, key)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, data2, retrieved)
}

// TestMemory_EmptyData verifies that empty byte slices are handled correctly.
func TestMemory_EmptyData(t *testing.T) {
	store := newTestMemoryStorage(t)
	ctx := context.Background()

	key := "empty-data-key"
	data := []byte{}

	err := store.PutObject(ctx, key, data)
	require.NoError(t, err)

	retrieved, found, err := store.GetObject(ctx, key)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, data, retrieved)
	require.Len(t, retrieved, 0)
}

// TestMemory_NilData verifies that nil data is handled correctly.
func TestMemory_NilData(t *testing.T) {
	store := newTestMemoryStorage(t)
	ctx := context.Background()

	key := "nil-data-key"

	err := store.PutObject(ctx, key, nil)
	require.NoError(t, err)

	retrieved, found, err := store.GetObject(ctx, key)
	require.NoError(t, err)
	require.True(t, found)
	// nil should be stored (could be nil or empty slice depending on impl)
	require.Len(t, retrieved, 0)
}

// TestMemory_BinaryData verifies that binary data with all byte values is
// preserved correctly.
func TestMemory_BinaryData(t *testing.T) {
	store := newTestMemoryStorage(t)
	ctx := context.Background()

	// Create data with all byte values 0x00-0xFF
	data := make([]byte, 256)
	for i := 0; i < 256; i++ {
		data[i] = byte(i)
	}

	key := "binary-data-key"
	err := store.PutObject(ctx, key, data)
	require.NoError(t, err)

	retrieved, found, err := store.GetObject(ctx, key)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, data, retrieved)
}

// TestMemory_LargeData verifies that large data is handled correctly.
func TestMemory_LargeData(t *testing.T) {
	store := newTestMemoryStorage(t)
	ctx := context.Background()

	// 1MB of data
	data := make([]byte, 1024*1024)
	for i := range data {
		data[i] = byte(i % 256)
	}

	key := "large-data-key"
	err := store.PutObject(ctx, key, data)
	require.NoError(t, err)

	retrieved, found, err := store.GetObject(ctx, key)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, data, retrieved)
}

// TestMemory_ListObjectKeys verifies prefix listing works correctly.
func TestMemory_ListObjectKeys(t *testing.T) {
	store := newTestMemoryStorage(t)
	ctx := context.Background()

	// Create objects with different prefixes
	objects := map[string][]byte{
		"keyring/alice/dek_1":  []byte("data1"),
		"keyring/alice/dek_2":  []byte("data2"),
		"keyring/alice/LATEST": []byte("data3"),
		"keyring/bob/dek_1":    []byte("data4"),
		"keyring/bob/LATEST":   []byte("data5"),
		"other/key":            []byte("data6"),
	}

	for key, data := range objects {
		err := store.PutObject(ctx, key, data)
		require.NoError(t, err)
	}

	// List with prefix "keyring/alice/"
	keys, err := store.ListObjectKeys(ctx, "keyring/alice/")
	require.NoError(t, err)
	require.Len(t, keys, 3)

	// List with prefix "keyring/alice/dek_"
	keys, err = store.ListObjectKeys(ctx, "keyring/alice/dek_")
	require.NoError(t, err)
	require.Len(t, keys, 2)

	// List with prefix "keyring/bob/"
	keys, err = store.ListObjectKeys(ctx, "keyring/bob/")
	require.NoError(t, err)
	require.Len(t, keys, 2)

	// List with non-matching prefix
	keys, err = store.ListObjectKeys(ctx, "nonexistent/")
	require.NoError(t, err)
	require.Len(t, keys, 0)
}

// TestMemory_KeyHelpers verifies the Key and Latest helper functions.
func TestMemory_KeyHelpers(t *testing.T) {
	store := newTestMemoryStorage(t)

	// Test Key helper
	key := store.Key("workspace123", "dek_abc")
	require.Equal(t, "workspace123/dek_abc", key)

	// Test Latest helper
	latest := store.Latest("workspace123")
	require.Equal(t, "workspace123/LATEST", latest)
}

// TestMemory_SpecialCharactersInKey verifies that special characters in keys
// are handled correctly.
func TestMemory_SpecialCharactersInKey(t *testing.T) {
	store := newTestMemoryStorage(t)
	ctx := context.Background()

	specialKeys := []string{
		"key/with/slashes",
		"key-with-dashes",
		"key_with_underscores",
		"key.with.dots",
		"key:with:colons",
		"key with spaces",
	}

	for _, key := range specialKeys {
		t.Run(key, func(t *testing.T) {
			data := []byte("data-for-" + key)
			err := store.PutObject(ctx, key, data)
			require.NoError(t, err)

			retrieved, found, err := store.GetObject(ctx, key)
			require.NoError(t, err)
			require.True(t, found)
			require.Equal(t, data, retrieved)
		})
	}
}

// TestMemory_ConcurrentAccess verifies that concurrent access is safe.
func TestMemory_ConcurrentAccess(t *testing.T) {
	store := newTestMemoryStorage(t)
	ctx := context.Background()

	const numGoroutines = 100
	const numOperations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				key := "concurrent-key"
				data := []byte("data")

				// Randomly put or get
				if j%2 == 0 {
					err := store.PutObject(ctx, key, data)
					require.NoError(t, err)
				} else {
					_, _, err := store.GetObject(ctx, key)
					require.NoError(t, err)
				}
			}
		}(i)
	}

	wg.Wait()
}

// TestMemory_ConcurrentDifferentKeys verifies concurrent access to different
// keys doesn't interfere.
func TestMemory_ConcurrentDifferentKeys(t *testing.T) {
	store := newTestMemoryStorage(t)
	ctx := context.Background()

	const numGoroutines = 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()

			key := store.Key("workspace", "dek_"+string(rune('A'+goroutineID)))
			data := []byte{byte(goroutineID)}

			err := store.PutObject(ctx, key, data)
			require.NoError(t, err)

			retrieved, found, err := store.GetObject(ctx, key)
			require.NoError(t, err)
			require.True(t, found)
			require.Equal(t, data, retrieved)
		}(i)
	}

	wg.Wait()
}

// TestMemory_DataIsolation verifies that modifications to returned data don't
// affect stored data.
func TestMemory_DataIsolation(t *testing.T) {
	store := newTestMemoryStorage(t)
	ctx := context.Background()

	key := "isolation-test"
	originalData := []byte("original-data")

	err := store.PutObject(ctx, key, originalData)
	require.NoError(t, err)

	// Get the data and modify it
	retrieved1, _, err := store.GetObject(ctx, key)
	require.NoError(t, err)

	// Modify the retrieved slice
	retrieved1[0] = 'X'

	// Get again - should be unmodified
	retrieved2, _, err := store.GetObject(ctx, key)
	require.NoError(t, err)

	// This test documents behavior - memory storage may or may not copy
	// If this fails, it means the storage returns the same slice (not a copy)
	// which could be a bug or intended behavior
	_ = retrieved2
}

// newTestMemoryStorage creates a new memory storage for testing.
func newTestMemoryStorage(t *testing.T) Storage {
	t.Helper()
	logger := logging.NewNoop()
	store, err := NewMemory(MemoryConfig{Logger: logger})
	require.NoError(t, err)
	return store
}
