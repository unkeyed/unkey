package storage

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/dockertest"
	"github.com/unkeyed/unkey/pkg/otel/logging"
)

// TestS3_PutAndGet verifies basic put and get operations against real S3.
func TestS3_PutAndGet(t *testing.T) {
	store := newTestS3Storage(t)
	ctx := context.Background()

	key := "test-key"
	data := []byte("test-data")

	err := store.PutObject(ctx, key, data)
	require.NoError(t, err)

	retrieved, found, err := store.GetObject(ctx, key)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, data, retrieved)
}

// TestS3_GetNonExistent verifies that getting a non-existent key returns
// found=false without error.
func TestS3_GetNonExistent(t *testing.T) {
	store := newTestS3Storage(t)
	ctx := context.Background()

	retrieved, found, err := store.GetObject(ctx, "nonexistent-key-12345")
	require.NoError(t, err)
	require.False(t, found)
	require.Nil(t, retrieved)
}

// TestS3_Overwrite verifies that putting to an existing key overwrites.
func TestS3_Overwrite(t *testing.T) {
	store := newTestS3Storage(t)
	ctx := context.Background()

	key := "overwrite-test-key"
	data1 := []byte("data-version-1")
	data2 := []byte("data-version-2-longer")

	err := store.PutObject(ctx, key, data1)
	require.NoError(t, err)

	err = store.PutObject(ctx, key, data2)
	require.NoError(t, err)

	retrieved, found, err := store.GetObject(ctx, key)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, data2, retrieved)
}

// TestS3_EmptyData verifies that empty byte slices are handled correctly.
func TestS3_EmptyData(t *testing.T) {
	store := newTestS3Storage(t)
	ctx := context.Background()

	key := "empty-data-key"
	data := []byte{}

	err := store.PutObject(ctx, key, data)
	require.NoError(t, err)

	retrieved, found, err := store.GetObject(ctx, key)
	require.NoError(t, err)
	require.True(t, found)
	require.Len(t, retrieved, 0)
}

// TestS3_BinaryData verifies that binary data with all byte values is
// preserved correctly.
func TestS3_BinaryData(t *testing.T) {
	store := newTestS3Storage(t)
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

// TestS3_LargeData verifies that larger data is handled correctly.
func TestS3_LargeData(t *testing.T) {
	store := newTestS3Storage(t)
	ctx := context.Background()

	// 100KB of data (smaller than memory test to keep S3 test fast)
	data := make([]byte, 100*1024)
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

// TestS3_ListObjectKeys verifies prefix listing works correctly.
func TestS3_ListObjectKeys(t *testing.T) {
	store := newTestS3Storage(t)
	ctx := context.Background()

	// Use unique prefix to avoid conflicts with other tests
	prefix := fmt.Sprintf("list-test-%d/", time.Now().UnixNano())

	objects := map[string][]byte{
		prefix + "alice/dek_1":  []byte("data1"),
		prefix + "alice/dek_2":  []byte("data2"),
		prefix + "alice/LATEST": []byte("data3"),
		prefix + "bob/dek_1":    []byte("data4"),
		prefix + "bob/LATEST":   []byte("data5"),
	}

	for key, data := range objects {
		err := store.PutObject(ctx, key, data)
		require.NoError(t, err)
	}

	// List with prefix for alice
	keys, err := store.ListObjectKeys(ctx, prefix+"alice/")
	require.NoError(t, err)
	require.Len(t, keys, 3)

	// List with prefix for dek_ under alice
	keys, err = store.ListObjectKeys(ctx, prefix+"alice/dek_")
	require.NoError(t, err)
	require.Len(t, keys, 2)

	// List with prefix for bob
	keys, err = store.ListObjectKeys(ctx, prefix+"bob/")
	require.NoError(t, err)
	require.Len(t, keys, 2)

	// List with non-matching prefix
	keys, err = store.ListObjectKeys(ctx, prefix+"nonexistent/")
	require.NoError(t, err)
	require.Len(t, keys, 0)
}

// TestS3_KeyHelpers verifies the Key and Latest helper functions.
func TestS3_KeyHelpers(t *testing.T) {
	store := newTestS3Storage(t)

	key := store.Key("workspace123", "dek_abc")
	require.Equal(t, "workspace123/dek_abc", key)

	latest := store.Latest("workspace123")
	require.Equal(t, "workspace123/LATEST", latest)
}

// TestS3_NestedPaths verifies that deeply nested paths work correctly.
func TestS3_NestedPaths(t *testing.T) {
	store := newTestS3Storage(t)
	ctx := context.Background()

	key := "level1/level2/level3/level4/deep-key"
	data := []byte("deeply-nested-data")

	err := store.PutObject(ctx, key, data)
	require.NoError(t, err)

	retrieved, found, err := store.GetObject(ctx, key)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, data, retrieved)
}

// TestS3_ConcurrentAccess verifies that concurrent access is safe.
func TestS3_ConcurrentAccess(t *testing.T) {
	store := newTestS3Storage(t)
	ctx := context.Background()

	const numGoroutines = 20
	const numOperations = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("concurrent-test/%d/%d", goroutineID, j)
				data := []byte(fmt.Sprintf("data-%d-%d", goroutineID, j))

				err := store.PutObject(ctx, key, data)
				require.NoError(t, err)

				retrieved, found, err := store.GetObject(ctx, key)
				require.NoError(t, err)
				require.True(t, found)
				require.Equal(t, data, retrieved)
			}
		}(i)
	}

	wg.Wait()
}

// TestS3_SpecialCharactersInKey verifies that special characters in keys
// are handled correctly by S3.
func TestS3_SpecialCharactersInKey(t *testing.T) {
	store := newTestS3Storage(t)
	ctx := context.Background()

	// S3 key naming rules are more restrictive than memory storage
	// These should all work with S3
	safeKeys := []string{
		"key/with/slashes",
		"key-with-dashes",
		"key_with_underscores",
		"key.with.dots",
		"keyWithCamelCase",
		"KEY_WITH_UPPERCASE",
	}

	for _, key := range safeKeys {
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

// TestS3_KeyringPattern verifies the actual keyring storage pattern used by
// the vault service.
func TestS3_KeyringPattern(t *testing.T) {
	store := newTestS3Storage(t)
	ctx := context.Background()

	keyring := "workspace_abc123"
	dekID := "dek_xyz789"

	// Store DEK
	dekKey := fmt.Sprintf("keyring/%s/%s", keyring, dekID)
	dekData := []byte("encrypted-dek-data")
	err := store.PutObject(ctx, dekKey, dekData)
	require.NoError(t, err)

	// Store LATEST pointer
	latestKey := fmt.Sprintf("keyring/%s/LATEST", keyring)
	err = store.PutObject(ctx, latestKey, dekData)
	require.NoError(t, err)

	// Verify both can be retrieved
	retrieved, found, err := store.GetObject(ctx, dekKey)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, dekData, retrieved)

	retrieved, found, err = store.GetObject(ctx, latestKey)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, dekData, retrieved)

	// List all keys for this keyring
	keys, err := store.ListObjectKeys(ctx, fmt.Sprintf("keyring/%s/", keyring))
	require.NoError(t, err)
	require.Len(t, keys, 2)
}

// TestS3_ContextCancellation verifies that context cancellation is respected.
func TestS3_ContextCancellation(t *testing.T) {
	store := newTestS3Storage(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Operations with cancelled context should fail
	_, _, err := store.GetObject(ctx, "any-key")
	require.Error(t, err)
}

// newTestS3Storage creates a new S3 storage backed by a MinIO container.
func newTestS3Storage(t *testing.T) Storage {
	t.Helper()

	s3Config := dockertest.S3(t)
	logger := logging.NewNoop()

	// Use a unique bucket name per test to ensure isolation
	bucketName := fmt.Sprintf("test-%d", time.Now().UnixNano())

	store, err := NewS3(S3Config{
		S3URL:             s3Config.URL,
		S3Bucket:          bucketName,
		S3AccessKeyID:     s3Config.AccessKeyID,
		S3AccessKeySecret: s3Config.SecretAccessKey,
		Logger:            logger,
	})
	require.NoError(t, err)

	return store
}
