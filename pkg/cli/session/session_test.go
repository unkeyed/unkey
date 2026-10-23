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
	"github.com/unkeyed/unkey/pkg/uid"
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
	workspaceA := uid.New(uid.WorkspacePrefix)
	workspaceB := uid.New(uid.WorkspacePrefix)
	sessionA := uid.New(uid.CLISessionPrefix)
	require.NoError(t, Save(Session{
		WorkspaceID:  workspaceA,
		SessionID:    sessionA,
		AccessToken:  "access_a",
		RefreshToken: "refresh_a",
		ExpiresAt:    1_900_000_000,
	}))
	require.NoError(t, Save(Session{
		WorkspaceID:  workspaceB,
		SessionID:    uid.New(uid.CLISessionPrefix),
		AccessToken:  "access_b",
		RefreshToken: "refresh_b",
		ExpiresAt:    1_900_000_000,
	}))

	active, err := Active()
	require.NoError(t, err)
	require.Equal(t, workspaceB, active.WorkspaceID)

	require.NoError(t, Switch(workspaceA))
	active, err = Active()
	require.NoError(t, err)
	require.Equal(t, workspaceA, active.WorkspaceID)

	removed, err := Delete(workspaceA)
	require.NoError(t, err)
	require.Equal(t, sessionA, removed.SessionID)
	active, err = Active()
	require.NoError(t, err)
	require.Equal(t, workspaceB, active.WorkspaceID)
}

func TestRefreshKeepsTheAPIURL(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	useMemoryStore(t)
	require.NoError(t, Save(Session{
		WorkspaceID:  uid.New(uid.WorkspacePrefix),
		SessionID:    uid.New(uid.CLISessionPrefix),
		AccessToken:  "access_a",
		RefreshToken: "refresh_a",
		ExpiresAt:    time.Now().Unix(),
		APIURL:       "http://127.0.0.1:8787",
	}))

	_, err := AccessToken(context.Background(), func(_ context.Context, current Session) (Session, error) {
		require.Equal(t, "http://127.0.0.1:8787", current.APIURL)
		return Session{
			AccessToken:  "access_b",
			RefreshToken: "refresh_b",
			ExpiresAt:    time.Now().Add(time.Hour).Unix(),
		}, nil
	})
	require.NoError(t, err)
	stored, err := Active()
	require.NoError(t, err)
	require.Equal(t, "http://127.0.0.1:8787", stored.APIURL)
	require.Equal(t, "refresh_b", stored.RefreshToken)
}

func TestKeychainSaveFailureKeepsTheRotatedTokenInTheFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	store := useMemoryStore(t)
	require.NoError(t, Save(Session{
		WorkspaceID:  uid.New(uid.WorkspacePrefix),
		SessionID:    uid.New(uid.CLISessionPrefix),
		AccessToken:  "access_a",
		RefreshToken: "refresh_a",
		ExpiresAt:    time.Now().Unix(),
	}))
	store.SetError(errors.New("keychain rejected the blob"))

	next, err := AccessToken(context.Background(), func(_ context.Context, current Session) (Session, error) {
		require.Equal(t, "refresh_a", current.RefreshToken)
		return Session{
			AccessToken:  "access_b",
			RefreshToken: "refresh_b",
			ExpiresAt:    time.Now().Add(time.Hour).Unix(),
		}, nil
	})
	require.NoError(t, err)
	require.Equal(t, "access_b", next.AccessToken)

	store.SetError(keystore.ErrUnavailable)
	stored, err := Active()
	require.NoError(t, err)
	require.Equal(t, "refresh_b", stored.RefreshToken)
}

func TestConcurrentRefreshDoesNotReuseTheToken(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	useMemoryStore(t)
	require.NoError(t, Save(Session{
		WorkspaceID:  uid.New(uid.WorkspacePrefix),
		SessionID:    uid.New(uid.CLISessionPrefix),
		AccessToken:  "access_a",
		RefreshToken: "refresh_a",
		ExpiresAt:    time.Now().Unix(),
	}))

	var calls atomic.Int32
	var mu sync.Mutex
	current := "refresh_a"
	refresh := func(_ context.Context, currentSession Session) (Session, error) {
		calls.Add(1)
		token := currentSession.RefreshToken
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
			used, err := AccessToken(context.Background(), refresh)
			tokens[i], errs[i] = used.AccessToken, err
		}()
	}
	wg.Wait()
	require.NoError(t, errs[0])
	require.NoError(t, errs[1])
	require.Equal(t, "access_b", tokens[0])
	require.Equal(t, "access_b", tokens[1])
	require.Equal(t, int32(1), calls.Load())
}
