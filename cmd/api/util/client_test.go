package util

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/cli"
)

func localJWT(t *testing.T, exp time.Time) string {
	t.Helper()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256"}`))
	payload, err := json.Marshal(map[string]any{"exp": exp.Unix()})
	require.NoError(t, err)
	return header + "." + base64.RawURLEncoding.EncodeToString(payload) + ".sig"
}

// resolveTokenForArgs builds a command with the standard client flags and runs
// resolveToken through real flag parsing.
func resolveTokenForArgs(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var token string
	var resolveErr error
	c := &cli.Command{
		Name: "test",
		Flags: []cli.Flag{
			RootKeyFlag(), APIURLFlag(), ConfigFlag(), WorkOSClientIDFlag(),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			token, resolveErr = resolveToken(cmd)
			return nil
		},
	}
	require.NoError(t, c.Run(context.Background(), append([]string{"test"}, args...)))
	return token, resolveErr
}

func TestResolveToken_ExplicitRootKeyWins(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	// A valid OAuth session is present, but the explicit root key must win.
	require.NoError(t, cli.SaveUserConfig(cli.UserConfig{
		AccessToken: "oauth-token",
		ExpiresAt:   time.Now().Add(time.Hour).Format(time.RFC3339),
	}))

	token, err := resolveTokenForArgs(t, "--root-key", "explicit_root_key")
	require.NoError(t, err)
	require.Equal(t, "explicit_root_key", token)
}

func TestResolveToken_ValidOAuthToken(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	// Any refresh attempt would hit this server and fail the test.
	t.Setenv("UNKEY_WORKOS_BASE_URL", "http://127.0.0.1:0")

	require.NoError(t, cli.SaveUserConfig(cli.UserConfig{
		AccessToken: "valid-access-token",
		ExpiresAt:   time.Now().Add(time.Hour).Format(time.RFC3339),
	}))

	token, err := resolveTokenForArgs(t)
	require.NoError(t, err)
	require.Equal(t, "valid-access-token", token)
}

func TestResolveToken_ExpiredRefreshesAndStores(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	newExp := time.Now().Add(time.Hour)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		require.Equal(t, "refresh_token", r.Form.Get("grant_type"))
		require.Equal(t, "old-refresh", r.Form.Get("refresh_token"))
		require.Equal(t, "org_1", r.Form.Get("organization_id"))
		w.Header().Set("Content-Type", "application/json")
		b, _ := json.Marshal(map[string]any{
			"access_token":    localJWT(t, newExp),
			"refresh_token":   "new-refresh",
			"organization_id": "org_1",
			"user":            map[string]any{"email": "james@unkey.com"},
		})
		fmt.Fprint(w, string(b))
	}))
	defer srv.Close()

	t.Setenv("UNKEY_WORKOS_BASE_URL", srv.URL)
	t.Setenv("UNKEY_WORKOS_CLIENT_ID", "client_abc")

	require.NoError(t, cli.SaveUserConfig(cli.UserConfig{
		AccessToken:  "expired-access",
		RefreshToken: "old-refresh",
		ExpiresAt:    time.Now().Add(-time.Minute).Format(time.RFC3339),
		OrgID:        "org_1",
	}))

	token, err := resolveTokenForArgs(t)
	require.NoError(t, err)
	require.Equal(t, localJWT(t, newExp), token)

	// The rotated tokens must be persisted.
	path, _ := cli.UserConfigPath()
	stored, err := cli.LoadUserConfig(path)
	require.NoError(t, err)
	require.Equal(t, cli.Secret("new-refresh"), stored.RefreshToken)
	require.Equal(t, "james@unkey.com", stored.UserEmail)
}

func TestResolveToken_RefreshFailsFallsBackToRootKey(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":"invalid_grant"}`)
	}))
	defer srv.Close()
	t.Setenv("UNKEY_WORKOS_BASE_URL", srv.URL)
	t.Setenv("UNKEY_WORKOS_CLIENT_ID", "client_abc")

	require.NoError(t, cli.SaveUserConfig(cli.UserConfig{
		RootKey:      "fallback_root_key",
		AccessToken:  "expired",
		RefreshToken: "dead-refresh",
		ExpiresAt:    time.Now().Add(-time.Minute).Format(time.RFC3339),
	}))

	token, err := resolveTokenForArgs(t)
	require.NoError(t, err)
	require.Equal(t, "fallback_root_key", token)
}

func TestResolveToken_RefreshFailsNoRootKey(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":"invalid_grant"}`)
	}))
	defer srv.Close()
	t.Setenv("UNKEY_WORKOS_BASE_URL", srv.URL)
	t.Setenv("UNKEY_WORKOS_CLIENT_ID", "client_abc")

	require.NoError(t, cli.SaveUserConfig(cli.UserConfig{
		AccessToken:  "expired",
		RefreshToken: "dead-refresh",
		ExpiresAt:    time.Now().Add(-time.Minute).Format(time.RFC3339),
	}))

	_, err := resolveTokenForArgs(t)
	require.Error(t, err)
	require.Contains(t, err.Error(), "session has expired")
}

func TestResolveToken_NoCredentials(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	_, err := resolveTokenForArgs(t)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no credentials found")
}
