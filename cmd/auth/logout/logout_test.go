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
	"github.com/unkeyed/unkey/pkg/cli/session"
	"github.com/unkeyed/unkey/pkg/uid"
)

type revokeRequest struct {
	SessionID string `json:"sessionId"`
}

type refreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type refreshData struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int64  `json:"expiresIn"`
	WorkspaceID  string `json:"workspaceId"`
	SessionID    string `json:"sessionId"`
}

type refreshResponse struct {
	Data refreshData `json:"data"`
}

type fakeAPI struct {
	mu           sync.Mutex
	revokeStatus int
	refreshes    []string
	revokes      []revokeRequest
	bearers      []string
}

func (f *fakeAPI) handler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		switch r.URL.Path {
		case "/v2/cli.exchangeToken":
			var req refreshRequest
			require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
			f.refreshes = append(f.refreshes, req.RefreshToken)
			w.Header().Set("content-type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(refreshResponse{Data: refreshData{
				AccessToken:  "access_refreshed",
				RefreshToken: "refresh_refreshed",
				ExpiresIn:    3600,
			}}))
		case "/v2/cli.revokeSession":
			var req revokeRequest
			require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
			f.revokes = append(f.revokes, req)
			f.bearers = append(f.bearers, r.Header.Get("authorization"))
			status := f.revokeStatus
			if status == 0 {
				status = http.StatusOK
			}
			w.WriteHeader(status)
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
	out := capture(t, &os.Stdout, func() {
		err = New().Run(context.Background(), []string{"unkey"})
	})
	require.NoError(t, err)
	require.Equal(t, []revokeRequest{{SessionID: current.SessionID}}, api.revokes)
	require.Equal(t, []string{"Bearer access_stored"}, api.bearers)
	require.Empty(t, api.refreshes)
	require.Contains(t, out, "Signed out of workspace "+current.WorkspaceID+".")
	require.Contains(t, out, "Active workspace is "+other.WorkspaceID+".")

	_, sessions, err := session.List()
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	require.Equal(t, other.WorkspaceID, sessions[0].WorkspaceID)
}

func TestLogoutRefreshesAnExpiredTokenBeforeRevoking(t *testing.T) {
	api := &fakeAPI{}
	srv := setup(t, api)
	current := saveSession(t, srv.URL, time.Now().Unix())

	var err error
	capture(t, &os.Stdout, func() {
		err = New().Run(context.Background(), []string{"unkey"})
	})
	require.NoError(t, err)
	require.Equal(t, []string{"refresh_stored"}, api.refreshes)
	require.Equal(t, []revokeRequest{{SessionID: current.SessionID}}, api.revokes)
	require.Equal(t, []string{"Bearer access_refreshed"}, api.bearers)

	_, err = session.Active()
	require.ErrorIs(t, err, session.ErrNotFound)
}

func TestLogoutDeletesTheSessionWhenRevokeFails(t *testing.T) {
	api := &fakeAPI{revokeStatus: http.StatusInternalServerError}
	srv := setup(t, api)
	current := saveSession(t, srv.URL, time.Now().Add(time.Hour).Unix())

	var err error
	stderr := capture(t, &os.Stderr, func() {
		capture(t, &os.Stdout, func() {
			err = New().Run(context.Background(), []string{"unkey"})
		})
	})
	require.ErrorContains(t, err, "revoke CLI session: revoke returned 500")
	require.Contains(t, stderr, "could not be revoked on the server")
	require.Equal(t, []revokeRequest{{SessionID: current.SessionID}}, api.revokes)

	_, err = session.Active()
	require.ErrorIs(t, err, session.ErrNotFound)
}

func TestLogoutWithoutASession(t *testing.T) {
	api := &fakeAPI{}
	setup(t, api)

	var err error
	out := capture(t, &os.Stdout, func() {
		err = New().Run(context.Background(), []string{"unkey"})
	})
	require.NoError(t, err)
	require.Contains(t, out, "No CLI session is stored.")
	require.Empty(t, api.revokes)
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
