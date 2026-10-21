// Package session stores one CLI login per workspace.
//
// The access token and refresh token live in the operating system secret store.
// When that store is unavailable they are written to ~/.unkey/sessions.json with
// mode 0600. A file left behind after a failed keychain write is newer than the
// keychain and replaces it on the next load.
package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/cli/keystore"
)

// ErrNotFound means there is no stored session for the active workspace.
var ErrNotFound = errors.New("cli session not found")

// Session is one workspace login.
type Session struct {
	WorkspaceID  string `json:"workspace_id"`
	SessionID    string `json:"session_id"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

type state struct {
	Active   string             `json:"active"`
	Sessions map[string]Session `json:"sessions"`
}

// Save stores session and makes its workspace the active one.
func Save(session Session) error {
	if session.WorkspaceID == "" || session.AccessToken == "" || session.RefreshToken == "" {
		return fmt.Errorf("cli session is incomplete")
	}
	current, err := read()
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
	current, err := read()
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
	current, err := read()
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
	current, err := read()
	if err != nil {
		return err
	}
	if _, ok := current.Sessions[workspaceID]; !ok {
		return ErrNotFound
	}
	current.Active = workspaceID
	return write(current)
}

// Delete removes one workspace session. It returns the removed session so the
// caller can revoke it.
func Delete(workspaceID string) (Session, error) {
	current, err := read()
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
	delete(current.Sessions, workspaceID)
	if current.Active == workspaceID {
		current.Active = ""
		for id := range current.Sessions {
			current.Active = id
			break
		}
	}
	if err := write(current); err != nil {
		return Session{}, err
	}
	return session, nil
}

// AccessToken returns a usable access token, refreshing it when it is within
// a minute of expiry. refresh exchanges the refresh token.
func AccessToken(ctx context.Context, refresh func(ctx context.Context, refreshToken string) (Session, error)) (string, error) {
	session, err := Active()
	if err != nil {
		return "", err
	}
	if session.ExpiresAt > time.Now().Add(time.Minute).Unix() {
		return session.AccessToken, nil
	}
	if refresh == nil {
		return "", fmt.Errorf("cli session expired")
	}
	next, err := refresh(ctx, session.RefreshToken)
	if err != nil {
		return "", err
	}
	if next.WorkspaceID == "" {
		next.WorkspaceID = session.WorkspaceID
	}
	if next.SessionID == "" {
		next.SessionID = session.SessionID
	}
	if err := Save(next); err != nil {
		return "", err
	}
	return next.AccessToken, nil
}

func read() (state, error) {
	if file, err := readFile(); err == nil {
		return file, nil
	} else if !errors.Is(err, os.ErrNotExist) && !errors.Is(err, ErrNotFound) {
		return state{}, err
	}
	raw, err := keystore.GetSessions()
	if err != nil {
		if errors.Is(err, keystore.ErrNotFound) || errors.Is(err, keystore.ErrUnavailable) {
			return state{}, ErrNotFound
		}
		return state{}, err
	}
	return decode(raw)
}

func write(current state) error {
	raw, err := json.Marshal(current)
	if err != nil {
		return err
	}
	err = keystore.SetSessions(string(raw))
	if err == nil {
		_ = removeFile()
		return nil
	}
	if !errors.Is(err, keystore.ErrUnavailable) {
		return err
	}
	return writeFile(raw)
}

func decode(raw string) (state, error) {
	var current state
	if strings.TrimSpace(raw) == "" {
		return state{}, ErrNotFound
	}
	if err := json.Unmarshal([]byte(raw), &current); err != nil {
		return state{}, err
	}
	if current.Sessions == nil {
		current.Sessions = map[string]Session{}
	}
	if len(current.Sessions) == 0 {
		return state{}, ErrNotFound
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
	if err != nil {
		return state{}, err
	}
	if setErr := keystore.SetSessions(string(raw)); setErr == nil {
		_ = os.Remove(path)
	}
	return current, nil
}

func writeFile(raw []byte) error {
	path, err := filePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o600)
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
