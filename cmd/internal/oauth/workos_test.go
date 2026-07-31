package oauth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// fakeAccessToken builds an unsigned JWT whose payload carries the given exp.
func fakeAccessToken(t *testing.T, exp time.Time) string {
	t.Helper()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	payload, err := json.Marshal(map[string]any{"exp": exp.Unix()})
	require.NoError(t, err)
	return header + "." + base64.RawURLEncoding.EncodeToString(payload) + ".sig"
}

func tokenResponseJSON(t *testing.T, accessToken, refresh, org, email string) string {
	t.Helper()
	b, err := json.Marshal(map[string]any{
		"access_token":    accessToken,
		"refresh_token":   refresh,
		"organization_id": org,
		"user":            map[string]any{"email": email},
	})
	require.NoError(t, err)
	return string(b)
}

func TestRequestDeviceAuthorization(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/user_management/authorize/device", r.URL.Path)
		require.NoError(t, r.ParseForm())
		require.Equal(t, "client_abc", r.Form.Get("client_id"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"device_code":"dc","user_code":"WDJB-MJHT","verification_uri":"https://work.os/device","verification_uri_complete":"https://work.os/device?code=WDJB-MJHT","expires_in":300,"interval":5}`)
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	got, err := c.RequestDeviceAuthorization(context.Background(), "client_abc")
	require.NoError(t, err)

	require.Equal(t, "dc", got.DeviceCode)
	require.Equal(t, "WDJB-MJHT", got.UserCode)
	require.Equal(t, "https://work.os/device?code=WDJB-MJHT", got.VerificationURIComplete)
	require.Equal(t, 300*time.Second, got.ExpiresIn)
	require.Equal(t, 5*time.Second, got.Interval)
}

func TestRequestDeviceAuthorization_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error":"server_error","error_description":"boom"}`)
	}))
	defer srv.Close()

	_, err := New(WithBaseURL(srv.URL)).RequestDeviceAuthorization(context.Background(), "client_abc")
	require.Error(t, err)
	require.Contains(t, err.Error(), "server_error")
}

func TestPollForToken_PendingThenSuccess(t *testing.T) {
	exp := time.Now().Add(5 * time.Minute)
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		require.Equal(t, deviceGrantType, r.Form.Get("grant_type"))
		require.Equal(t, "dc", r.Form.Get("device_code"))
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `{"error":"authorization_pending"}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, tokenResponseJSON(t, fakeAccessToken(t, exp), "refresh", "org_1", "james@unkey.com"))
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL))
	got, err := c.PollForToken(context.Background(), "client_abc", "dc", 5*time.Millisecond, time.Minute)
	require.NoError(t, err)
	require.Equal(t, int32(2), calls.Load())
	require.Equal(t, "refresh", got.RefreshToken)
	require.Equal(t, "org_1", got.OrganizationID)
	require.Equal(t, "james@unkey.com", got.Email)
	require.WithinDuration(t, exp, got.ExpiresAt, time.Second)
}

func TestPollForToken_SlowDownIncreasesInterval(t *testing.T) {
	exp := time.Now().Add(5 * time.Minute)
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `{"error":"slow_down"}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, tokenResponseJSON(t, fakeAccessToken(t, exp), "refresh", "org_1", "e@x.com"))
	}))
	defer srv.Close()

	// Base interval 5ms, slow-down increment 20ms → the gap before the second
	// poll must be at least 25ms.
	c := New(WithBaseURL(srv.URL), WithSlowDownIncrement(20*time.Millisecond))
	start := time.Now()
	_, err := c.PollForToken(context.Background(), "client_abc", "dc", 5*time.Millisecond, time.Minute)
	require.NoError(t, err)
	require.GreaterOrEqual(t, time.Since(start), 25*time.Millisecond)
	require.Equal(t, int32(2), calls.Load())
}

func TestPollForToken_AccessDenied(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":"access_denied"}`)
	}))
	defer srv.Close()

	_, err := New(WithBaseURL(srv.URL)).PollForToken(context.Background(), "c", "dc", time.Millisecond, time.Minute)
	require.ErrorIs(t, err, ErrAccessDenied)
}

func TestPollForToken_ExpiredByDeadline(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":"authorization_pending"}`)
	}))
	defer srv.Close()

	// Deadline already passed → one poll, then expiry.
	_, err := New(WithBaseURL(srv.URL)).PollForToken(context.Background(), "c", "dc", time.Millisecond, time.Nanosecond)
	require.ErrorIs(t, err, ErrExpiredToken)
}

func TestPollForToken_ExpiredTokenError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":"expired_token"}`)
	}))
	defer srv.Close()

	_, err := New(WithBaseURL(srv.URL)).PollForToken(context.Background(), "c", "dc", time.Millisecond, time.Minute)
	require.ErrorIs(t, err, ErrExpiredToken)
}

func TestPollForToken_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":"authorization_pending"}`)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := New(WithBaseURL(srv.URL)).PollForToken(ctx, "c", "dc", time.Hour, time.Hour)
	require.ErrorIs(t, err, context.Canceled)
}

func TestRefresh_IncludesOrgID(t *testing.T) {
	exp := time.Now().Add(5 * time.Minute)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		require.Equal(t, "refresh_token", r.Form.Get("grant_type"))
		require.Equal(t, "rt", r.Form.Get("refresh_token"))
		require.Equal(t, "org_9", r.Form.Get("organization_id"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, tokenResponseJSON(t, fakeAccessToken(t, exp), "rt2", "org_9", "e@x.com"))
	}))
	defer srv.Close()

	got, err := New(WithBaseURL(srv.URL)).Refresh(context.Background(), "c", "rt", "org_9")
	require.NoError(t, err)
	require.Equal(t, "rt2", got.RefreshToken)
	require.Equal(t, "org_9", got.OrganizationID)
}

func TestRefresh_OmitsOrgIDWhenEmpty(t *testing.T) {
	exp := time.Now().Add(5 * time.Minute)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		require.False(t, r.Form.Has("organization_id"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, tokenResponseJSON(t, fakeAccessToken(t, exp), "rt2", "", "e@x.com"))
	}))
	defer srv.Close()

	_, err := New(WithBaseURL(srv.URL)).Refresh(context.Background(), "c", "rt", "")
	require.NoError(t, err)
}

func TestRefresh_InvalidGrant(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":"invalid_grant","error_description":"refresh token expired"}`)
	}))
	defer srv.Close()

	_, err := New(WithBaseURL(srv.URL)).Refresh(context.Background(), "c", "rt", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid_grant")
}

func TestAccessTokenExpiry(t *testing.T) {
	exp := time.Unix(1_800_000_000, 0).UTC()
	got, err := accessTokenExpiry(fakeAccessToken(t, exp))
	require.NoError(t, err)
	require.Equal(t, exp, got)

	_, err = accessTokenExpiry("not-a-jwt")
	require.Error(t, err)
}
