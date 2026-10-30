package logout

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/internal/sdk/models/components"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/cli/session"
	"github.com/unkeyed/unkey/pkg/uid"
)

type fakeAPI struct {
	mu           sync.Mutex
	revokeStatus int
	refreshed    session.Session
	refreshes    []string
	revokes      []components.V2CliRevokeSessionRequestBody
	bearers      []string
}

func (f *fakeAPI) handler(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		switch r.URL.Path {
		case "/v2/cli.exchangeToken":
			var req components.V2CliExchangeTokenRequestBody2
			require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
			require.NotEmpty(t, req.RefreshToken)
			require.Nil(t, req.LoginID)
			f.refreshes = append(f.refreshes, req.RefreshToken)
			w.Header().Set("content-type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(components.V2CliExchangeTokenResponseBody{
				Data: components.V2CliExchangeTokenResponseData{
					AccessToken:  "access_refreshed",
					RefreshToken: "refresh_refreshed",
					ExpiresIn:    3600,
					WorkspaceID:  f.refreshed.WorkspaceID,
					SessionID:    f.refreshed.SessionID,
				},
				Meta: components.Meta{RequestID: uid.New(uid.RequestPrefix)},
			}))
		case "/v2/cli.revokeSession":
			var req components.V2CliRevokeSessionRequestBody
			require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
			f.revokes = append(f.revokes, req)
			f.bearers = append(f.bearers, r.Header.Get("authorization"))
			status := f.revokeStatus
			if status == 0 {
				status = http.StatusOK
			}
			w.Header().Set("content-type", "application/json")
			w.WriteHeader(status)
			if status == http.StatusOK {
				require.NoError(t, json.NewEncoder(w).Encode(components.V2CliRevokeSessionResponseBody{
					Data: components.EmptyResponse{},
					Meta: components.Meta{RequestID: uid.New(uid.RequestPrefix)},
				}))
			}
		default:
			http.NotFound(w, r)
		}
	}
}

func setup(t *testing.T, api *fakeAPI) *httptest.Server {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("UNKEY_API_BASE_URL", "")
	session.SetStore(&session.MemoryStore{})
	t.Cleanup(func() { session.SetStore(nil) })
	srv := httptest.NewServer(api.handler(t))
	t.Cleanup(srv.Close)
	return srv
}

func saveSession(t *testing.T, apiURL string, expiresAt int64) session.Session {
	t.Helper()
	stored := session.Session{
		WorkspaceID:  uid.New(uid.WorkspacePrefix),
		SessionID:    uid.New(uid.CLISessionPrefix),
		AccessToken:  "access_stored",
		RefreshToken: "refresh_stored",
		ExpiresAt:    expiresAt,
		APIURL:       apiURL,
	}
	require.NoError(t, session.Save(stored))
	return stored
}

func TestLogoutRevokesAndDeletesTheActiveSession(t *testing.T) {
	api := &fakeAPI{}
	srv := setup(t, api)
	other := saveSession(t, srv.URL, time.Now().Add(time.Hour).Unix())
	current := saveSession(t, srv.URL, time.Now().Add(time.Hour).Unix())

	var err error
	var stdout string
	stderr := capture(t, &os.Stderr, func() {
		stdout = capture(t, &os.Stdout, func() {
			err = New().Run(context.Background(), []string{"unkey"})
		})
	})
	require.NoError(t, err)
	require.Equal(t, []components.V2CliRevokeSessionRequestBody{{SessionID: current.SessionID}}, api.revokes)
	require.Equal(t, []string{"Bearer access_stored"}, api.bearers)
	require.Empty(t, api.refreshes)
	require.Equal(t, "Signed out.\n", stdout)
	require.Equal(t, "Switched to workspace "+other.WorkspaceID+".\n", stderr)

	active, sessions, err := session.List()
	require.NoError(t, err)
	require.Equal(t, other.WorkspaceID, active)
	require.Len(t, sessions, 1)
	require.Equal(t, other.WorkspaceID, sessions[0].WorkspaceID)
}

func TestLogoutRefreshesAnExpiredTokenBeforeRevoking(t *testing.T) {
	api := &fakeAPI{}
	srv := setup(t, api)
	current := saveSession(t, srv.URL, time.Now().Unix())
	api.refreshed = current

	var err error
	var stdout string
	stderr := capture(t, &os.Stderr, func() {
		stdout = capture(t, &os.Stdout, func() {
			err = New().Run(context.Background(), []string{"unkey"})
		})
	})
	require.NoError(t, err)
	require.Equal(t, "Signed out.\n", stdout)
	require.Empty(t, stderr)
	require.Equal(t, []string{"refresh_stored"}, api.refreshes)
	require.Equal(t, []components.V2CliRevokeSessionRequestBody{{SessionID: current.SessionID}}, api.revokes)
	require.Equal(t, []string{"Bearer access_refreshed"}, api.bearers)

	_, err = session.Active()
	require.ErrorIs(t, err, session.ErrNotFound)
}

func TestLogoutDeletesTheSessionWhenRevokeFails(t *testing.T) {
	for _, test := range []struct {
		name         string
		otherSession bool
	}{
		{name: "last session"},
		{name: "another saved session", otherSession: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			api := &fakeAPI{revokeStatus: http.StatusInternalServerError}
			srv := setup(t, api)
			var other session.Session
			if test.otherSession {
				other = saveSession(t, srv.URL, time.Now().Add(time.Hour).Unix())
			}
			current := saveSession(t, srv.URL, time.Now().Add(time.Hour).Unix())

			var err error
			var stdout string
			stderr := capture(t, &os.Stderr, func() {
				stdout = capture(t, &os.Stdout, func() {
					err = New().Run(context.Background(), []string{"unkey"})
				})
			})
			require.ErrorContains(t, err, "could not sign out: server returned HTTP 500")
			require.Empty(t, stdout)
			require.Contains(t, stderr, "The session was removed from this machine but could not be revoked on the server. It stays valid until it expires.\n")
			require.NotContains(t, stderr, "Signed out")
			require.Equal(t, []components.V2CliRevokeSessionRequestBody{{SessionID: current.SessionID}}, api.revokes)

			active, err := session.Active()
			if test.otherSession {
				require.NoError(t, err)
				require.Equal(t, other, active)
				require.Contains(t, stderr, "Switched to workspace "+other.WorkspaceID+".\n")
			} else {
				require.ErrorIs(t, err, session.ErrNotFound)
				require.NotContains(t, stderr, "Switched to workspace")
			}
		})
	}
}

func TestLogoutWithoutASession(t *testing.T) {
	api := &fakeAPI{}
	setup(t, api)
	t.Setenv("UNKEY_ROOT_KEY", "root_key_from_environment")
	require.NoError(t, cli.SaveUserConfig(cli.UserConfig{RootKey: "root_key_from_config"}))

	var err error
	var stdout string
	stderr := capture(t, &os.Stderr, func() {
		stdout = capture(t, &os.Stdout, func() {
			err = New().Run(context.Background(), []string{"unkey"})
		})
	})
	require.NoError(t, err)
	require.Equal(t, "No saved browser session.\n", stdout)
	require.Empty(t, stderr)
	require.Empty(t, api.revokes)
	require.Empty(t, api.refreshes)
	require.Equal(t, "root_key_from_environment", os.Getenv("UNKEY_ROOT_KEY"))
	path, err := cli.UserConfigPath()
	require.NoError(t, err)
	config, err := cli.LoadUserConfig(path)
	require.NoError(t, err)
	require.Equal(t, "root_key_from_config", config.RootKey)
}

func capture(t *testing.T, target **os.File, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	original := *target
	*target = w
	t.Cleanup(func() { *target = original })
	fn()
	*target = original
	require.NoError(t, w.Close())
	out, err := io.ReadAll(r)
	require.NoError(t, err)
	return string(out)
}
