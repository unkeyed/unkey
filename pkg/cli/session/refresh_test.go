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
	"github.com/unkeyed/unkey/internal/sdk"
	"github.com/unkeyed/unkey/internal/sdk/models/apierrors"
	"github.com/unkeyed/unkey/internal/sdk/models/components"
	"github.com/unkeyed/unkey/pkg/uid"
)

func newRefreshServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

func writeRefreshResponse[T any](t *testing.T, w http.ResponseWriter, status int, body T) {
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
		var req components.V2CliExchangeTokenRequestBody2
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		require.Equal(t, "refresh_a", req.RefreshToken)
		require.Nil(t, req.LoginID)
		writeRefreshResponse(t, w, http.StatusOK, components.V2CliExchangeTokenResponseBody{
			Data: components.V2CliExchangeTokenResponseData{
				AccessToken:  "access_b",
				RefreshToken: "refresh_b",
				ExpiresIn:    3600,
				WorkspaceID:  workspaceID,
				SessionID:    sessionID,
			},
			Meta: components.Meta{RequestID: uid.New(uid.RequestPrefix)},
		})
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

func TestRefreshRejectsIncompleteExchangesWithoutRetrying(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*components.V2CliExchangeTokenResponseData)
	}{
		{name: "empty data", change: func(data *components.V2CliExchangeTokenResponseData) {
			*data = components.V2CliExchangeTokenResponseData{}
		}},
		{name: "empty access token", change: func(data *components.V2CliExchangeTokenResponseData) { data.AccessToken = "" }},
		{name: "empty refresh token", change: func(data *components.V2CliExchangeTokenResponseData) { data.RefreshToken = "" }},
		{name: "empty workspace", change: func(data *components.V2CliExchangeTokenResponseData) { data.WorkspaceID = "" }},
		{name: "empty session", change: func(data *components.V2CliExchangeTokenResponseData) { data.SessionID = "" }},
		{name: "zero expiry", change: func(data *components.V2CliExchangeTokenResponseData) { data.ExpiresIn = 0 }},
		{name: "negative expiry", change: func(data *components.V2CliExchangeTokenResponseData) { data.ExpiresIn = -1 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := components.V2CliExchangeTokenResponseData{
				AccessToken:  "access_b",
				RefreshToken: "refresh_b",
				ExpiresIn:    3600,
				WorkspaceID:  uid.New(uid.WorkspacePrefix),
				SessionID:    uid.New(uid.CLISessionPrefix),
			}
			tc.change(&data)
			var calls atomic.Int32
			srv := newRefreshServer(t, func(w http.ResponseWriter, _ *http.Request) {
				calls.Add(1)
				writeRefreshResponse(t, w, http.StatusOK, components.V2CliExchangeTokenResponseBody{
					Data: data,
					Meta: components.Meta{RequestID: uid.New(uid.RequestPrefix)},
				})
			})

			next, err := Refresh(t.Context(), srv.URL, "refresh_a")
			require.Error(t, err)
			require.Empty(t, next)
			require.Equal(t, int32(1), calls.Load())
		})
	}
}

func TestRefreshRejectsPendingAndMissingSuccessBodiesWithoutRetrying(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   any
	}{
		{
			name: "pending", status: http.StatusAccepted,
			body: components.V2CliExchangeTokenPendingResponseBody{
				Data: components.EmptyResponse{}, Meta: components.Meta{RequestID: uid.New(uid.RequestPrefix)},
			},
		},
		{
			name: "pending with tokens", status: http.StatusAccepted,
			body: components.V2CliExchangeTokenResponseBody{
				Data: components.V2CliExchangeTokenResponseData{
					AccessToken: "access_b", RefreshToken: "refresh_b", ExpiresIn: 3600,
					WorkspaceID: uid.New(uid.WorkspacePrefix), SessionID: uid.New(uid.CLISessionPrefix),
				},
				Meta: components.Meta{RequestID: uid.New(uid.RequestPrefix)},
			},
		},
		{name: "no content", status: http.StatusNoContent},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			srv := newRefreshServer(t, func(w http.ResponseWriter, _ *http.Request) {
				calls.Add(1)
				if tc.body == nil {
					w.WriteHeader(tc.status)
					return
				}
				w.Header().Set("Retry-After", "2")
				writeRefreshResponse(t, w, tc.status, tc.body)
			})

			next, err := Refresh(t.Context(), srv.URL, "refresh_a")
			require.Error(t, err)
			require.Empty(t, next)
			require.Equal(t, int32(1), calls.Load())
		})
	}
}

func TestRefreshReportsTheErrorDetail(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		detail string
	}{
		{name: "revoked", status: http.StatusUnauthorized, detail: "The refresh token was revoked."},
		{name: "denied", status: http.StatusForbidden, detail: "Authorization was denied."},
		{name: "expired", status: http.StatusGone, detail: "This session expired."},
		{name: "rate limited", status: http.StatusTooManyRequests, detail: "Try again later."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			srv := newRefreshServer(t, func(w http.ResponseWriter, _ *http.Request) {
				calls.Add(1)
				writeRefreshResponse(t, w, tc.status, apierrors.UnauthorizedErrorResponse{
					Error_: components.BaseError{Detail: tc.detail, Status: int64(tc.status)},
					Meta:   components.Meta{RequestID: uid.New(uid.RequestPrefix)},
				})
			})

			_, err := Refresh(context.Background(), srv.URL, "refresh_a")
			var response *sdk.ResponseError
			require.ErrorAs(t, err, &response)
			require.Equal(t, tc.status, response.StatusCode)
			require.EqualError(t, err, tc.detail)
			require.Equal(t, int32(1), calls.Load())
		})
	}
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
		var req components.V2CliExchangeTokenRequestBody2
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		require.Equal(t, "refresh_a", req.RefreshToken)
		require.Nil(t, req.LoginID)
		if calls.Add(1) == 1 {
			w.Header().Set("content-type", "text/html")
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte("<html>bad gateway</html>"))
			return
		}
		writeRefreshResponse(t, w, http.StatusOK, components.V2CliExchangeTokenResponseBody{
			Data: components.V2CliExchangeTokenResponseData{
				AccessToken:  "access_b",
				RefreshToken: "refresh_b",
				ExpiresIn:    3600,
				WorkspaceID:  uid.New(uid.WorkspacePrefix),
				SessionID:    uid.New(uid.CLISessionPrefix),
			},
			Meta: components.Meta{RequestID: uid.New(uid.RequestPrefix)},
		})
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
