package keyring

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/vault/internal/storage"
	"github.com/unkeyed/unkey/svc/vault/keys"
)

func TestDeletedEnvironmentSweepRemovesLateWritesAfterWriterFailure(t *testing.T) {
	for _, tc := range []struct {
		name   string
		target string
		rewrap bool
	}{
		{name: "create-versioned", target: "versioned"},
		{name: "create-latest", target: "LATEST"},
		{name: "rewrap-versioned", target: "versioned", rewrap: true},
		{name: "rewrap-latest", target: "LATEST", rewrap: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			writer := setupTestKeyring(t)
			id := uid.New(uid.EnvironmentPrefix)
			write := func() error {
				_, err := writer.CreateKey(t.Context(), id)
				return err
			}
			if tc.rewrap {
				dek, err := writer.CreateKey(t.Context(), id)
				require.NoError(t, err)
				kek, _, err := keys.GenerateMasterKey()
				require.NoError(t, err)
				writer.encryptionKey = kek
				writer.decryptionKeys[kek.GetId()] = kek
				keyID := dek.GetId()
				if tc.target == "LATEST" {
					keyID = "LATEST"
				}
				write = func() error { return writer.RewrapKey(t.Context(), "keyring/"+id+"/"+keyID) }
			}
			store := &delayedKeyStorage{
				Storage: writer.store,
				prefix:  "keyring/" + id + "/",
				target:  tc.target,
				entered: make(chan struct{}),
				resume:  make(chan struct{}),
			}
			writer.store = store
			cleaner, err := New(Config{Store: store, EncryptionKey: writer.encryptionKey, DecryptionKeys: writer.decryptionKeys})
			require.NoError(t, err)
			var resume sync.Once
			release := func() { resume.Do(func() { close(store.resume) }) }
			done := make(chan error, 1)
			t.Cleanup(func() {
				release()
				for range done {
				}
			})
			go func() {
				defer close(done)
				done <- write()
			}()
			select {
			case <-store.entered:
			case err := <-done:
				t.Fatalf("writer returned before persisting key: %v", err)
			case <-t.Context().Done():
				t.Fatal(t.Context().Err())
			}
			require.NoError(t, cleaner.DeleteEnvironment(t.Context(), id))
			require.NoError(t, cleaner.SweepDeletedEnvironments(t.Context()))
			release()
			require.ErrorIs(t, <-done, errInterruptedWrite)
			remaining, err := store.ListObjectKeys(t.Context(), store.prefix)
			require.NoError(t, err)
			require.Len(t, remaining, 1, "the acknowledged deletion can be followed by a late durable write")
			require.NoError(t, cleaner.SweepDeletedEnvironments(t.Context()))
			remaining, err = store.ListObjectKeys(t.Context(), store.prefix)
			require.NoError(t, err)
			require.Empty(t, remaining)
			_, err = cleaner.GetOrCreateKey(t.Context(), id, "LATEST")
			require.ErrorIs(t, err, ErrDeleted)
		})
	}
}

func TestDeletedEnvironmentSweepContinuesAfterOneRingFails(t *testing.T) {
	kr := setupTestKeyring(t)
	blockedID, otherID := uid.New(uid.EnvironmentPrefix), uid.New(uid.EnvironmentPrefix)
	_, err := kr.CreateKey(t.Context(), blockedID)
	require.NoError(t, err)
	_, err = kr.CreateKey(t.Context(), otherID)
	require.NoError(t, err)
	store := &failedKeyDeletionStorage{Storage: kr.store, blockedPrefix: "keyring/" + blockedID + "/"}
	kr.store = store
	require.Error(t, kr.DeleteEnvironment(t.Context(), blockedID))
	require.NoError(t, kr.DeleteEnvironment(t.Context(), otherID))
	require.NoError(t, store.Storage.PutObject(t.Context(), "keyring/"+otherID+"/LATEST", []byte("late write")))
	require.Error(t, kr.SweepDeletedEnvironments(t.Context()))
	remaining, err := store.ListObjectKeys(t.Context(), "keyring/"+otherID+"/")
	require.NoError(t, err)
	require.Empty(t, remaining)
	store.blockedPrefix = ""
	require.NoError(t, kr.SweepDeletedEnvironments(t.Context()))
	remaining, err = store.ListObjectKeys(t.Context(), "keyring/"+blockedID+"/")
	require.NoError(t, err)
	require.Empty(t, remaining)
}

var errInterruptedWrite = errors.New("writer stopped after persisting key")

type delayedKeyStorage struct {
	storage.Storage
	prefix  string
	target  string
	entered chan struct{}
	resume  chan struct{}
}

func (s *delayedKeyStorage) PutObject(ctx context.Context, key string, data []byte) error {
	keyID, matches := strings.CutPrefix(key, s.prefix)
	if !matches || (s.target == "LATEST") != (keyID == "LATEST") {
		return s.Storage.PutObject(ctx, key, data)
	}
	close(s.entered)
	select {
	case <-s.resume:
	case <-ctx.Done():
		return ctx.Err()
	}
	if err := s.Storage.PutObject(ctx, key, data); err != nil {
		return err
	}
	return errInterruptedWrite
}

type failedKeyDeletionStorage struct {
	storage.Storage
	blockedPrefix string
}

func (s *failedKeyDeletionStorage) DeleteObject(ctx context.Context, key string) error {
	if s.blockedPrefix != "" && strings.HasPrefix(key, s.blockedPrefix) {
		return errors.New("delete denied")
	}
	return s.Storage.DeleteObject(ctx, key)
}
