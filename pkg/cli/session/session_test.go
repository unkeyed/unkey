package session

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/cli/keystore"
)

func useMemoryStore(t *testing.T) *MemoryStore {
	t.Helper()
	store := &MemoryStore{}
	SetStore(store)
	t.Cleanup(func() { SetStore(nil) })
	return store
}

func TestSaveSwitchAndDelete(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	useMemoryStore(t)
	require.NoError(t, Save(Session{
		WorkspaceID:  "ws_a",
		SessionID:    "clis_a",
		AccessToken:  "access_a",
		RefreshToken: "refresh_a",
		ExpiresAt:    1_900_000_000,
	}))
	require.NoError(t, Save(Session{
		WorkspaceID:  "ws_b",
		SessionID:    "clis_b",
		AccessToken:  "access_b",
		RefreshToken: "refresh_b",
		ExpiresAt:    1_900_000_000,
	}))

	active, err := Active()
	require.NoError(t, err)
	require.Equal(t, "ws_b", active.WorkspaceID)

	require.NoError(t, Switch("ws_a"))
	active, err = Active()
	require.NoError(t, err)
	require.Equal(t, "ws_a", active.WorkspaceID)

	removed, err := Delete("ws_a")
	require.NoError(t, err)
	require.Equal(t, "clis_a", removed.SessionID)
	active, err = Active()
	require.NoError(t, err)
	require.Equal(t, "ws_b", active.WorkspaceID)
}

func TestKeychainSaveFailureKeepsTheRotatedTokenInTheFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	store := useMemoryStore(t)
	require.NoError(t, Save(Session{
		WorkspaceID:  "ws_a",
		SessionID:    "clis_a",
		AccessToken:  "access_a",
		RefreshToken: "refresh_a",
		ExpiresAt:    time.Now().Unix(),
	}))
	store.SetError(errors.New("keychain rejected the blob"))

	next, err := AccessToken(context.Background(), func(_ context.Context, refreshToken string) (Session, error) {
		require.Equal(t, "refresh_a", refreshToken)
		return Session{
			AccessToken:  "access_b",
			RefreshToken: "refresh_b",
			ExpiresAt:    time.Now().Add(time.Hour).Unix(),
		}, nil
	})
	require.NoError(t, err)
	require.Equal(t, "access_b", next)

	store.SetError(keystore.ErrUnavailable)
	stored, err := Active()
	require.NoError(t, err)
	require.Equal(t, "refresh_b", stored.RefreshToken)
}

func TestConcurrentRefreshDoesNotReuseTheToken(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	useMemoryStore(t)
	require.NoError(t, Save(Session{
		WorkspaceID:  "ws_a",
		SessionID:    "clis_a",
		AccessToken:  "access_a",
		RefreshToken: "refresh_a",
		ExpiresAt:    time.Now().Unix(),
	}))

	var calls atomic.Int32
	var mu sync.Mutex
	current := "refresh_a"
	refresh := func(_ context.Context, token string) (Session, error) {
		calls.Add(1)
		mu.Lock()
		defer mu.Unlock()
		if token != current {
			return Session{}, errors.New("refresh token was already used")
		}
		current = "refresh_b"
		time.Sleep(30 * time.Millisecond)
		return Session{
			AccessToken:  "access_b",
			RefreshToken: "refresh_b",
			ExpiresAt:    time.Now().Add(time.Hour).Unix(),
		}, nil
	}

	var wg sync.WaitGroup
	errs := make([]error, 2)
	tokens := make([]string, 2)
	for i := range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tokens[i], errs[i] = AccessToken(context.Background(), refresh)
		}()
	}
	wg.Wait()
	require.NoError(t, errs[0])
	require.NoError(t, errs[1])
	require.Equal(t, "access_b", tokens[0])
	require.Equal(t, "access_b", tokens[1])
	require.Equal(t, int32(1), calls.Load())
}
