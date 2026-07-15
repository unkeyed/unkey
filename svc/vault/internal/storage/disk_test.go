package storage

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDisk_PutAndGet(t *testing.T) {
	store := newTestDiskStorage(t)
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

func TestDisk_GetNonExistent(t *testing.T) {
	store := newTestDiskStorage(t)
	ctx := context.Background()

	retrieved, found, err := store.GetObject(ctx, "nonexistent-key")
	require.NoError(t, err)
	require.False(t, found)
	require.Nil(t, retrieved)
}

func TestDisk_Overwrite(t *testing.T) {
	store := newTestDiskStorage(t)
	ctx := context.Background()

	key := "test-key"
	data1 := []byte("data-version-1")
	data2 := []byte("data-version-2-longer")

	require.NoError(t, store.PutObject(ctx, key, data1))
	require.NoError(t, store.PutObject(ctx, key, data2))

	retrieved, found, err := store.GetObject(ctx, key)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, data2, retrieved)
}

func TestDisk_EmptyData(t *testing.T) {
	store := newTestDiskStorage(t)
	ctx := context.Background()

	key := "empty-data-key"
	require.NoError(t, store.PutObject(ctx, key, []byte{}))

	retrieved, found, err := store.GetObject(ctx, key)
	require.NoError(t, err)
	require.True(t, found)
	require.Len(t, retrieved, 0)
}

func TestDisk_BinaryData(t *testing.T) {
	store := newTestDiskStorage(t)
	ctx := context.Background()

	data := make([]byte, 256)
	for i := range data {
		data[i] = byte(i)
	}

	key := "binary-data-key"
	require.NoError(t, store.PutObject(ctx, key, data))

	retrieved, found, err := store.GetObject(ctx, key)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, data, retrieved)
}

func TestDisk_LargeData(t *testing.T) {
	store := newTestDiskStorage(t)
	ctx := context.Background()

	data := make([]byte, 1024*1024)
	for i := range data {
		data[i] = byte(i % 256)
	}

	key := "large-data-key"
	require.NoError(t, store.PutObject(ctx, key, data))

	retrieved, found, err := store.GetObject(ctx, key)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, data, retrieved)
}

func TestDisk_ListObjectKeys(t *testing.T) {
	store := newTestDiskStorage(t)
	ctx := context.Background()

	objects := map[string][]byte{
		"keyring/alice/dek_1":  []byte("data1"),
		"keyring/alice/dek_2":  []byte("data2"),
		"keyring/alice/LATEST": []byte("data3"),
		"keyring/bob/dek_1":    []byte("data4"),
		"keyring/bob/LATEST":   []byte("data5"),
		"other/key":            []byte("data6"),
	}

	for key, data := range objects {
		require.NoError(t, store.PutObject(ctx, key, data))
	}

	keys, err := store.ListObjectKeys(ctx, "keyring/alice/")
	require.NoError(t, err)
	require.Len(t, keys, 3)

	keys, err = store.ListObjectKeys(ctx, "keyring/alice/dek_")
	require.NoError(t, err)
	require.Len(t, keys, 2)

	keys, err = store.ListObjectKeys(ctx, "keyring/bob/")
	require.NoError(t, err)
	require.Len(t, keys, 2)

	keys, err = store.ListObjectKeys(ctx, "nonexistent/")
	require.NoError(t, err)
	require.Len(t, keys, 0)
}

func TestDisk_KeyHelpers(t *testing.T) {
	store := newTestDiskStorage(t)

	require.Equal(t, "workspace123/dek_abc", store.Key("workspace123", "dek_abc"))
	require.Equal(t, "workspace123/LATEST", store.Latest("workspace123"))
}

func TestDisk_RejectsPathTraversal(t *testing.T) {
	store := newTestDiskStorage(t)
	ctx := context.Background()

	bad := []string{
		"../escape",
		"foo/../../escape",
		"/abs/path",
	}
	for _, key := range bad {
		t.Run(key, func(t *testing.T) {
			err := store.PutObject(ctx, key, []byte("nope"))
			require.Error(t, err)
		})
	}
}

func TestDisk_PersistsAcrossInstances(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	first, err := NewDisk(dir)
	require.NoError(t, err)
	require.NoError(t, first.PutObject(ctx, "keyring/ws/LATEST", []byte("hello")))

	second, err := NewDisk(dir)
	require.NoError(t, err)

	got, found, err := second.GetObject(ctx, "keyring/ws/LATEST")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, []byte("hello"), got)
}

func TestDisk_ConcurrentDifferentKeys(t *testing.T) {
	store := newTestDiskStorage(t)
	ctx := context.Background()

	const numGoroutines = 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			key := store.Key("workspace", fmt.Sprintf("dek_%d", id))
			data := []byte{byte(id)}

			assert.NoError(t, store.PutObject(ctx, key, data))

			retrieved, found, err := store.GetObject(ctx, key)
			assert.NoError(t, err)
			assert.True(t, found)
			assert.Equal(t, data, retrieved)
		}(i)
	}

	wg.Wait()
}

func newTestDiskStorage(t *testing.T) Storage {
	t.Helper()
	store, err := NewDisk(t.TempDir())
	require.NoError(t, err)
	return store
}
