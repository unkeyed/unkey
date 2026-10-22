package storage

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestDeleteObjectIsIdempotentAndIsolated(t *testing.T) {
	for name, create := range map[string]func(*testing.T) Storage{
		"memory": newTestMemoryStorage,
		"disk":   newTestDiskStorage,
		"s3":     newTestS3Storage,
	} {
		t.Run(name, func(t *testing.T) {
			store := create(t)
			key := "keyring/" + uid.New(uid.TestPrefix) + "/LATEST"
			other := key + "-other"
			require.NoError(t, store.PutObject(t.Context(), key, []byte("delete")))
			require.NoError(t, store.PutObject(t.Context(), other, []byte("keep")))
			for range 2 {
				require.NoError(t, store.DeleteObject(t.Context(), key))
			}
			_, found, err := store.GetObject(t.Context(), key)
			require.NoError(t, err)
			require.False(t, found)
			data, found, err := store.GetObject(t.Context(), other)
			require.NoError(t, err)
			require.True(t, found)
			require.Equal(t, []byte("keep"), data)
		})
	}
}
