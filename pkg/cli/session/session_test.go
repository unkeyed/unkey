package session

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
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

	removed, err := Delete(workspaceA, sessionA)
	require.NoError(t, err)
	require.Equal(t, sessionA, removed.SessionID)
	active, err = Active()
	require.NoError(t, err)
	require.Equal(t, workspaceB, active.WorkspaceID)
}

func TestSaveRoundTripsALargeSession(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	useMemoryStore(t)
	session := largeSession()
	require.NoError(t, Save(session))

	active, err := Active()
	require.NoError(t, err)
	require.Equal(t, session, active)
}

func TestLogoutDoesNotRestoreAKeychainSession(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	store := useMemoryStore(t)
	current := largeSession()
	current.WorkspaceID = uid.New(uid.WorkspacePrefix)
	current.SessionID = uid.New(uid.CLISessionPrefix)
	require.NoError(t, Save(current))
	store.SetError(keystore.ErrUnavailable)
	_, err := Delete(current.WorkspaceID, current.SessionID)
	require.NoError(t, err)
	store.SetError(nil)
	_, err = Active()
	require.ErrorIs(t, err, ErrNotFound)
	_, sessions, err := List()
	require.NoError(t, err)
	require.Empty(t, sessions)
}

func TestDeletePreservesAReplacementSession(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	useMemoryStore(t)
	current := largeSession()
	current.WorkspaceID = uid.New(uid.WorkspacePrefix)
	current.SessionID = uid.New(uid.CLISessionPrefix)
	require.NoError(t, Save(current))
	next := current
	next.SessionID = uid.New(uid.CLISessionPrefix)
	require.NoError(t, Save(next))
	_, err := Delete(current.WorkspaceID, current.SessionID)
	require.ErrorIs(t, err, ErrNotFound)
	stored, err := Active()
	require.NoError(t, err)
	require.Equal(t, next, stored)
}

type obstructingStore struct {
	MemoryStore
	path string
}

func (s *obstructingStore) Set(secret string) error {
	if err := os.MkdirAll(s.path, 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(s.path, "blocked"), nil, 0o600); err != nil {
		return err
	}
	return s.MemoryStore.Set(secret)
}

func TestSaveReportsAnUnusableFallback(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	SetStore(&obstructingStore{path: filepath.Join(home, ".unkey", "sessions.json")})
	t.Cleanup(func() { SetStore(nil) })
	err := Save(Session{
		WorkspaceID:  uid.New(uid.WorkspacePrefix),
		SessionID:    uid.New(uid.CLISessionPrefix),
		AccessToken:  "access",
		RefreshToken: "refresh",
	})
	require.Error(t, err)
}

func TestKeychainFailureWritesAPrivateSessionFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	store := useMemoryStore(t)
	store.SetError(keystore.ErrUnavailable)
	session := largeSession()
	require.NoError(t, Save(session))

	path := filepath.Join(home, ".unkey", "sessions.json")
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Zero(t, info.Mode().Perm()&0o077)
	entries, err := os.ReadDir(filepath.Dir(path))
	require.NoError(t, err)
	for _, entry := range entries {
		require.False(t, strings.HasPrefix(entry.Name(), ".sessions-"))
	}

	active, err := Active()
	require.NoError(t, err)
	require.Equal(t, session, active)
}

func TestCorruptSessionFileDoesNotHideTheKeychain(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	useMemoryStore(t)
	session := Session{
		WorkspaceID:  "ws_1",
		SessionID:    "clis_1",
		AccessToken:  "access",
		RefreshToken: "refresh",
		ExpiresAt:    1_900_000_000,
	}
	require.NoError(t, Save(session))
	path := filepath.Join(home, ".unkey", "sessions.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte(`{"active":"ws_1","sessions":{"ws_1":{"access_token":"abc"`), 0o600))

	active, err := Active()
	require.NoError(t, err)
	require.Equal(t, session, active)
	_, statErr := os.Stat(path)
	require.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestCorruptSessionFileWithNoKeychainIsNotFound(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	useMemoryStore(t)
	path := filepath.Join(home, ".unkey", "sessions.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte(`{"active":"ws_1","sessions":{"ws_1":{"access_token":"abc"`), 0o600))

	_, err := Active()
	require.ErrorIs(t, err, ErrNotFound)
	require.NotContains(t, err.Error(), "unexpected end of JSON input")
	_, statErr := os.Stat(path)
	require.ErrorIs(t, statErr, os.ErrNotExist)

	require.NoError(t, Save(Session{
		WorkspaceID:  "ws_1",
		SessionID:    "clis_1",
		AccessToken:  "access",
		RefreshToken: "refresh",
		ExpiresAt:    1_900_000_000,
	}))
	active, err := Active()
	require.NoError(t, err)
	require.Equal(t, "ws_1", active.WorkspaceID)
}

func TestCorruptKeychainTellsTheUserToLogInAgain(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	store := useMemoryStore(t)
	require.NoError(t, store.Set(`{"active":"ws_1"`))

	_, err := Active()
	require.ErrorIs(t, err, ErrNotFound)
	require.ErrorContains(t, err, "run unkey login again")
	require.NotContains(t, err.Error(), "unexpected end of JSON input")
	_, getErr := store.Get()
	require.ErrorIs(t, getErr, keystore.ErrNotFound)
}

func largeSession() Session {
	return Session{
		WorkspaceID:  "ws_large",
		SessionID:    "clis_large",
		AccessToken:  strings.Repeat("a", 5000),
		RefreshToken: strings.Repeat("b", 2000),
		ExpiresAt:    1_900_000_000,
	}
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
