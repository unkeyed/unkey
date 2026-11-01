package status

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/internal/sdk/models/components"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/cli/session"
	"github.com/unkeyed/unkey/pkg/uid"
)

const (
	accessToken  = "access_secret_fixture"
	refreshToken = "refresh_secret_fixture"
	rootKey      = "unkey_root_secret_fixture"
)

type fakeAPI struct {
	mu            sync.Mutex
	listStatus    int
	refreshStatus int
	userID        string
	sessionID     string
	bearers       []string
	refreshes     int
}

func (f *fakeAPI) handler(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		w.Header().Set("content-type", "application/json")
		switch r.URL.Path {
		case "/v2/cli.listSessions":
			f.bearers = append(f.bearers, r.Header.Get("authorization"))
			if f.listStatus != 0 {
				w.WriteHeader(f.listStatus)
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"detail": "The CLI session is not valid.", "status": f.listStatus}}))
				return
			}
			require.NoError(t, json.NewEncoder(w).Encode(components.V2CliListSessionsResponseBody{
				Meta: components.Meta{RequestID: uid.New(uid.RequestPrefix)},
				Data: components.V2CliListSessionsResponseData{Sessions: []components.V2CliListSessionsSession{
					{ID: uid.New(uid.CLISessionPrefix), UserID: uid.New("user"), Permissions: []string{}},
					{ID: f.sessionID, UserID: f.userID, Permissions: []string{}},
				}},
			}))
		case "/v2/cli.exchangeToken":
			f.refreshes++
			w.WriteHeader(f.refreshStatus)
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"detail": "The refresh token is not valid.", "status": f.refreshStatus}}))
		default:
			http.NotFound(w, r)
		}
	}
}

func setup(t *testing.T, api *fakeAPI) *httptest.Server {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("UNKEY_API_BASE_URL", "")
	t.Setenv("UNKEY_ROOT_KEY", "")
	t.Setenv("UNKEY_CONFIG", "")
	t.Setenv("UNKEY_OUTPUT", "")
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
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		APIURL:       apiURL,
	}
	require.NoError(t, session.Save(stored))
	return stored
}

func runStatus(t *testing.T, args ...string) (report, string, string, error) {
	t.Helper()
	var err error
	var stdout string
	stderr := capture(t, &os.Stderr, func() {
		stdout = capture(t, &os.Stdout, func() {
			err = New().Run(context.Background(), append([]string{"unkey", "--json"}, args...))
		})
	})
	for _, secret := range []string{accessToken, refreshToken, rootKey} {
		require.NotContains(t, stdout, secret)
		require.NotContains(t, stderr, secret)
		if err != nil {
			require.NotContains(t, err.Error(), secret)
		}
	}
	var out struct {
		Data report `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &out), stdout)
	return out.Data, stdout, stderr, err
}

func TestStatusVerifiesTheBrowserSession(t *testing.T) {
	api := &fakeAPI{userID: uid.New("user")}
	srv := setup(t, api)
	current := saveSession(t, srv.URL, time.Now().Add(time.Hour).Unix())
	api.sessionID = current.SessionID

	got, _, stderr, err := runStatus(t)
	require.NoError(t, err)
	require.Empty(t, stderr)
	require.Equal(t, report{
		Credential:     "browser session",
		Authentication: "verified",
		APIURL:         srv.URL,
		WorkspaceID:    current.WorkspaceID,
		SessionID:      current.SessionID,
		UserID:         api.userID,
		Detail:         "",
	}, got)
	require.Equal(t, []string{"Bearer " + accessToken}, api.bearers)
}

func TestStatusReportsARejectedBrowserSession(t *testing.T) {
	for _, test := range []struct {
		name          string
		expiresAt     int64
		listStatus    int
		refreshStatus int
		refreshes     int
		checks        int
	}{
		{name: "access token rejected", expiresAt: time.Now().Add(time.Hour).Unix(), listStatus: http.StatusUnauthorized, checks: 1},
		{name: "refresh token rejected", expiresAt: time.Now().Unix(), refreshStatus: http.StatusUnauthorized, refreshes: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			api := &fakeAPI{listStatus: test.listStatus, refreshStatus: test.refreshStatus}
			srv := setup(t, api)
			current := saveSession(t, srv.URL, test.expiresAt)

			got, _, _, err := runStatus(t)
			require.ErrorContains(t, err, "Run 'unkey login' to sign in again")
			require.Equal(t, "browser session", got.Credential)
			require.Equal(t, "failed", got.Authentication)
			require.Equal(t, current.WorkspaceID, got.WorkspaceID)
			require.NotEmpty(t, got.Detail)
			require.Empty(t, got.UserID)
			require.Len(t, api.bearers, test.checks)
			require.Equal(t, test.refreshes, api.refreshes)
		})
	}
}

func TestStatusDoesNotCheckRootKeys(t *testing.T) {
	for _, test := range []struct {
		name       string
		source     util.CredentialSource
		env        string
		configFlag bool
		configEnv  bool
		args       []string
	}{
		{name: "flag", source: util.SourceRootKeyFlag, args: []string{"--root-key", rootKey}},
		{name: "environment", source: util.SourceRootKeyEnv, env: rootKey},
		{name: "explicit config", source: util.SourceConfig, configFlag: true},
		{name: "saved root key", source: util.SourceSavedRootKey, configEnv: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			api := &fakeAPI{}
			srv := setup(t, api)
			t.Setenv("UNKEY_ROOT_KEY", test.env)
			args := test.args
			path := filepath.Join(t.TempDir(), "config.toml")
			require.NoError(t, os.WriteFile(path, []byte("root_key = \""+rootKey+"\"\n"), 0o600))
			if test.configEnv {
				t.Setenv("UNKEY_CONFIG", path)
			} else {
				saveSession(t, srv.URL, time.Now().Add(time.Hour).Unix())
			}
			if test.configFlag {
				args = []string{"--config", path}
			}

			got, _, _, err := runStatus(t, args...)
			require.NoError(t, err)
			require.Equal(t, string(test.source), got.Credential)
			require.Equal(t, "unchecked", got.Authentication)
			require.Empty(t, got.WorkspaceID)
			require.Empty(t, got.UserID)
			require.Empty(t, api.bearers)
		})
	}
}

func TestStatusWithoutCredentials(t *testing.T) {
	api := &fakeAPI{}
	setup(t, api)
	t.Setenv("UNKEY_CONFIG", filepath.Join(t.TempDir(), "missing.toml"))

	got, _, _, err := runStatus(t)
	require.ErrorIs(t, err, util.ErrNoCredentials)
	require.Equal(t, "none", got.Credential)
	require.Equal(t, "missing", got.Authentication)
	require.Equal(t, "https://api.unkey.com", got.APIURL)
}

func TestStatusIsAvailableAsWhoami(t *testing.T) {
	api := &fakeAPI{}
	setup(t, api)
	t.Setenv("UNKEY_ROOT_KEY", rootKey)
	root := &cli.Command{Name: "unkey", Commands: []*cli.Command{{Name: "auth", Commands: []*cli.Command{New()}}}}

	stdout := capture(t, &os.Stdout, func() {
		require.NoError(t, root.Run(context.Background(), []string{"unkey", "whoami"}))
	})
	require.Contains(t, stdout, "Credential:      UNKEY_ROOT_KEY\n")
	require.NotContains(t, stdout, rootKey)
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
