package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
)

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

type refreshError struct {
	Detail string `json:"detail"`
}

type refreshResponse struct {
	Data  *refreshData  `json:"data,omitempty"`
	Error *refreshError `json:"error,omitempty"`
}

func newRefreshServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

func writeRefreshResponse(t *testing.T, w http.ResponseWriter, status int, body refreshResponse) {
	t.Helper()
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	require.NoError(t, json.NewEncoder(w).Encode(body))
}

func TestRefreshReturnsTheNewPair(t *testing.T) {
	workspaceID := uid.New(uid.WorkspacePrefix)
	sessionID := uid.New(uid.CLISessionPrefix)
	srv := newRefreshServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v2/cli.exchangeToken", r.URL.Path)
		var req refreshRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		require.Equal(t, "refresh_a", req.RefreshToken)
		writeRefreshResponse(t, w, http.StatusOK, refreshResponse{Data: &refreshData{
			AccessToken:  "access_b",
			RefreshToken: "refresh_b",
			ExpiresIn:    3600,
			WorkspaceID:  workspaceID,
			SessionID:    sessionID,
		}})
	})

	before := time.Now()
	next, err := Refresh(context.Background(), srv.URL+"/", "refresh_a")
	require.NoError(t, err)
	require.Equal(t, "access_b", next.AccessToken)
	require.Equal(t, "refresh_b", next.RefreshToken)
	require.Equal(t, workspaceID, next.WorkspaceID)
	require.Equal(t, sessionID, next.SessionID)
	require.Equal(t, srv.URL, next.APIURL)
	require.GreaterOrEqual(t, next.ExpiresAt, before.Add(time.Hour).Unix())
	require.LessOrEqual(t, next.ExpiresAt, time.Now().Add(time.Hour).Unix())
}

func TestRefreshReportsTheErrorDetail(t *testing.T) {
	var calls atomic.Int32
	srv := newRefreshServer(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		writeRefreshResponse(t, w, http.StatusUnauthorized, refreshResponse{Error: &refreshError{Detail: "The refresh token was revoked."}})
	})

	_, err := Refresh(context.Background(), srv.URL, "refresh_a")
	require.EqualError(t, err, "The refresh token was revoked.")
	require.Equal(t, int32(1), calls.Load())
}

func TestRefreshRejectsANonJSONBody(t *testing.T) {
	srv := newRefreshServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("content-type", "text/html")
		_, _ = w.Write([]byte("<html>ok</html>"))
	})

	_, err := Refresh(context.Background(), srv.URL, "refresh_a")
	require.EqualError(t, err, "refresh returned 200")
}

func TestRefreshRetriesAServerErrorWithTheSameToken(t *testing.T) {
	var calls atomic.Int32
	srv := newRefreshServer(t, func(w http.ResponseWriter, r *http.Request) {
		var req refreshRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		require.Equal(t, "refresh_a", req.RefreshToken)
		if calls.Add(1) == 1 {
			w.Header().Set("content-type", "text/html")
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte("<html>bad gateway</html>"))
			return
		}
		writeRefreshResponse(t, w, http.StatusOK, refreshResponse{Data: &refreshData{
			AccessToken:  "access_b",
			RefreshToken: "refresh_b",
			ExpiresIn:    3600,
			WorkspaceID:  uid.New(uid.WorkspacePrefix),
			SessionID:    uid.New(uid.CLISessionPrefix),
		}})
	})

	next, err := Refresh(context.Background(), srv.URL, "refresh_a")
	require.NoError(t, err)
	require.Equal(t, "refresh_b", next.RefreshToken)
	require.Equal(t, int32(2), calls.Load())
}

func TestRefreshRetriesATimeoutOnce(t *testing.T) {
	original := refreshAttemptTimeout
	refreshAttemptTimeout = 50 * time.Millisecond
	t.Cleanup(func() { refreshAttemptTimeout = original })

	var calls atomic.Int32
	srv := newRefreshServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		select {
		case <-r.Context().Done():
		case <-time.After(time.Second):
		}
	})

	_, err := Refresh(context.Background(), srv.URL, "refresh_a")
	require.Error(t, err)
	require.Equal(t, int32(2), calls.Load())
}
