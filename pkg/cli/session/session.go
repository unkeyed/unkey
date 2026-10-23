// Package session stores one CLI login per workspace.
//
// The access token and refresh token live in the operating system secret store.
// When that store is unavailable they are written to ~/.unkey/sessions.json with
// mode 0600. A file left behind after a failed keychain write is newer than the
// keychain and replaces it on the next load. A keychain blob that is not valid
// session JSON is removed and reported as missing so a truncated item does not
// block the next login. A corrupt sessions.json is removed and does not hide a
// keychain session.
package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/unkeyed/unkey/pkg/cli/keystore"
)

// Store is the secret-store seam for the session blob. Tests pass a
// [MemoryStore] so they do not read or write the operating system keychain.
type Store interface {
	Get() (string, error)
	Set(secret string) error
	Delete() error
}

type keychainStore struct{}

func (keychainStore) Get() (string, error) {
	return keystore.GetSessions()
}

func (keychainStore) Set(secret string) error {
	return keystore.SetSessions(secret)
}

func (keychainStore) Delete() error {
	return keystore.DeleteSessions()
}

// MemoryStore keeps the session blob in memory.
type MemoryStore struct {
	mu     sync.Mutex
	value  string
	getErr error
	setErr error
}

// SetError makes the next Set call fail.
func (m *MemoryStore) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.setErr = err
}

func (m *MemoryStore) Get() (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.getErr != nil {
		return "", m.getErr
	}
	if m.value == "" {
		return "", keystore.ErrNotFound
	}
	return m.value, nil
}

func (m *MemoryStore) Set(secret string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.setErr != nil {
		return m.setErr
	}
	m.value = secret
	return nil
}

func (m *MemoryStore) Delete() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.value = ""
	return nil
}

var secrets Store = keychainStore{}

// SetStore replaces the secret store. Passing nil restores the keychain.
func SetStore(store Store) {
	if store == nil {
		secrets = keychainStore{}
		return
	}
	secrets = store
}

// ErrNotFound means there is no stored session for the active workspace.
var ErrNotFound = errors.New("cli session not found")

// DefaultAPIURL is the API a CLI session belongs to when none was stored.
const DefaultAPIURL = "https://api.unkey.com"

// NormalizeAPIURL trims space and a trailing slash so two spellings of the
// same host compare equal.
func NormalizeAPIURL(raw string) string {
	return strings.TrimRight(strings.TrimSpace(raw), "/")
}

// errCorruptSession marks a stored blob that is not session JSON. It is wrapped
// with ErrNotFound so a later login can replace it, and the text tells the
// user to sign in again.
var errCorruptSession = errors.New("stored CLI session is unreadable")

// Session is one workspace login.
type Session struct {
	WorkspaceID  string `json:"workspace_id"`
	SessionID    string `json:"session_id"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
	APIURL       string `json:"api_url,omitempty"`
}

type state struct {
	Active   string             `json:"active"`
	Sessions map[string]Session `json:"sessions"`
}

// Save stores session and makes its workspace the active one.
func Save(session Session) error {
	unlock, err := lockSessions()
	if err != nil {
		return err
	}
	defer unlock()
	return save(session)
}

func save(session Session) error {
	if session.WorkspaceID == "" || session.AccessToken == "" || session.RefreshToken == "" {
		return fmt.Errorf("cli session is incomplete")
	}
	current, err := readLocked()
	if err != nil && !errors.Is(err, ErrNotFound) {
		return err
	}
	if current.Sessions == nil {
		current.Sessions = map[string]Session{}
	}
	current.Sessions[session.WorkspaceID] = session
	current.Active = session.WorkspaceID
	return write(current)
}

// Active returns the session for the active workspace.
func Active() (Session, error) {
	unlock, err := lockSessions()
	if err != nil {
		return Session{}, err
	}
	defer unlock()
	return active()
}

func active() (Session, error) {
	current, err := readLocked()
	if err != nil {
		return Session{}, err
	}
	session, ok := current.Sessions[current.Active]
	if !ok {
		return Session{}, ErrNotFound
	}
	return session, nil
}

// List returns every stored session and the active workspace id.
func List() (active string, sessions []Session, err error) {
	unlock, err := lockSessions()
	if err != nil {
		return "", nil, err
	}
	defer unlock()
	current, err := readLocked()
	if errors.Is(err, ErrNotFound) {
		return "", nil, nil
	}
	if err != nil {
		return "", nil, err
	}
	out := make([]Session, 0, len(current.Sessions))
	for _, session := range current.Sessions {
		out = append(out, session)
	}
	return current.Active, out, nil
}

// Switch changes the active workspace. The workspace must already have a session.
func Switch(workspaceID string) error {
	unlock, err := lockSessions()
	if err != nil {
		return err
	}
	defer unlock()
	current, err := readLocked()
	if err != nil {
		return err
	}
	if _, ok := current.Sessions[workspaceID]; !ok {
		return ErrNotFound
	}
	current.Active = workspaceID
	return write(current)
}

// Delete removes the workspace session only if its ID still matches sessionID.
func Delete(workspaceID, sessionID string) (Session, error) {
	unlock, err := lockSessions()
	if err != nil {
		return Session{}, err
	}
	defer unlock()
	current, err := readLocked()
	if err != nil {
		return Session{}, err
	}
	session, ok := current.Sessions[workspaceID]
	if !ok {
		if workspaceID == "" {
			session, ok = current.Sessions[current.Active]
			workspaceID = current.Active
		}
		if !ok {
			return Session{}, ErrNotFound
		}
	}
	if session.SessionID != sessionID {
		return Session{}, ErrNotFound
	}
	delete(current.Sessions, workspaceID)
	if current.Active == workspaceID {
		remaining := slices.Sorted(maps.Keys(current.Sessions))
		current.Active = ""
		if len(remaining) > 0 {
			current.Active = remaining[0]
		}
	}
	if err := write(current); err != nil {
		return Session{}, err
	}
	return session, nil
}

// AccessToken returns the active session with a usable access token,
// refreshing it when it is within a minute of expiry. refresh exchanges the
// refresh token. The returned session is the one the token belongs to, which
// may differ from an earlier [Active] call if the workspace was switched.
//
// The read, refresh, and save run under a file lock. The session is read
// again after the lock is acquired, so a caller that waited does not present
// a refresh token another caller already rotated.
func AccessToken(ctx context.Context, refresh func(ctx context.Context, current Session) (Session, error)) (Session, error) {
	unlock, err := lockSessions()
	if err != nil {
		return Session{}, err
	}
	defer unlock()

	session, err := active()
	if err != nil {
		return Session{}, err
	}
	if session.ExpiresAt > time.Now().Add(time.Minute).Unix() {
		return session, nil
	}
	if refresh == nil {
		return Session{}, fmt.Errorf("cli session expired")
	}
	next, err := refresh(ctx, session)
	if err != nil {
		return Session{}, err
	}
	if next.WorkspaceID == "" {
		next.WorkspaceID = session.WorkspaceID
	}
	if next.SessionID == "" {
		next.SessionID = session.SessionID
	}
	if next.APIURL == "" {
		next.APIURL = session.APIURL
	}
	if err := save(next); err != nil {
		return Session{}, err
	}
	return next, nil
}

func readLocked() (state, error) {
	file, err := readFile()
	if err == nil {
		return file, nil
	}
	// readFile already removed a corrupt sessions.json. Fall through so that
	// file cannot hide a session still stored in the keychain.
	if !errors.Is(err, os.ErrNotExist) && !errors.Is(err, ErrNotFound) && !errors.Is(err, errCorruptSession) {
		return state{}, err
	}
	raw, err := secrets.Get()
	if err != nil {
		if errors.Is(err, keystore.ErrNotFound) || errors.Is(err, keystore.ErrUnavailable) {
			return state{}, ErrNotFound
		}
		return state{}, err
	}
	current, err := decode(raw)
	if errors.Is(err, errCorruptSession) {
		_ = secrets.Delete()
		return state{}, err
	}
	return current, err
}

func write(current state) error {
	raw, err := json.Marshal(current)
	if err != nil {
		return err
	}
	if err = secrets.Set(string(raw)); err == nil {
		err = removeFile()
		if err == nil {
			return nil
		}
	}
	if fileErr := writeFile(raw); fileErr != nil {
		return errors.Join(err, fileErr)
	}
	return nil
}

func decode(raw string) (state, error) {
	var current state
	if strings.TrimSpace(raw) == "" {
		return state{}, ErrNotFound
	}
	if err := json.Unmarshal([]byte(raw), &current); err != nil {
		return state{}, fmt.Errorf("%w: %w; run unkey login again", ErrNotFound, errCorruptSession)
	}
	if current.Sessions == nil {
		current.Sessions = map[string]Session{}
	}
	return current, nil
}

func filePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".unkey", "sessions.json"), nil
}

func readFile() (state, error) {
	path, err := filePath()
	if err != nil {
		return state{}, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return state{}, err
	}
	current, err := decode(string(raw))
	if errors.Is(err, errCorruptSession) {
		_ = os.Remove(path)
		return state{}, err
	}
	if err != nil {
		return state{}, err
	}
	if setErr := secrets.Set(string(raw)); setErr == nil {
		_ = os.Remove(path)
	}
	return current, nil
}

func writeFile(raw []byte) error {
	path, err := filePath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".sessions-*.json")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return nil
}

func removeFile() error {
	path, err := filePath()
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
