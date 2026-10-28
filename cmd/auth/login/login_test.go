package login

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/internal/sdk"
	"github.com/unkeyed/unkey/internal/sdk/models/apierrors"
	"github.com/unkeyed/unkey/internal/sdk/models/components"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/cli/credentials"
	"github.com/unkeyed/unkey/pkg/cli/keystore"
	"github.com/unkeyed/unkey/pkg/cli/session"
	"github.com/unkeyed/unkey/pkg/uid"
)

func useSessionStore(t *testing.T) *session.MemoryStore {
	t.Helper()
	store := &session.MemoryStore{}
	session.SetStore(store)
	t.Cleanup(func() { session.SetStore(nil) })
	return store
}

func TestFormatUserCode(t *testing.T) {
	require.Equal(t, "Z H V G - S X Q M", formatUserCode("zhvg-sxqm"))
	require.Equal(t, "Z H V G - S X Q M", formatUserCode("  ZHVG-SXQM  "))
}

func TestRootKeyFlagStoresTheKey(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("UNKEY_ROOT_KEY", "")
	store := &memoryRootKeyStore{}

	out := captureStdout(t, func() {
		cmd := newCommand(store)
		err := cmd.Run(context.Background(), []string{"unkey", "--root-key", "unkey_from_flag"})
		require.NoError(t, err)
	})
	require.Contains(t, out, "Authentication successful. Key stored in the system keychain.")
	require.Equal(t, "unkey_from_flag", store.secret)
	require.NotContains(t, configBody(t, home), "unkey_from_flag")
}

func TestRootKeyFlagFallsBackToTheConfigFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("UNKEY_ROOT_KEY", "")
	store := &memoryRootKeyStore{unavailable: true}

	path := filepath.Join(home, ".unkey", "config.toml")
	out := captureStdout(t, func() {
		cmd := newCommand(store)
		err := cmd.Run(context.Background(), []string{"unkey", "--root-key", "unkey_from_flag"})
		require.NoError(t, err)
	})
	require.Contains(t, out, "Authentication successful. Key stored in "+path)

	cfg, err := cli.LoadUserConfig(path)
	require.NoError(t, err)
	require.Equal(t, "unkey_from_flag", cfg.RootKey)
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestDeviceLoginPollsUntilTheKeyIsIssued(t *testing.T) {
	useSessionStore(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	store := &memoryRootKeyStore{}
	loginID := uid.New(uid.DeviceAuthorizationPrefix)
	workspaceID := uid.New(uid.WorkspacePrefix)
	sessionID := uid.New(uid.CLISessionPrefix)

	var mu sync.Mutex
	polls := 0
	var startedBody components.V2CliStartDeviceLoginRequestBody
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/cli.startDeviceLogin":
			require.NoError(t, json.NewDecoder(r.Body).Decode(&startedBody))
			writeEnvelope(t, w, http.StatusOK, startedEnvelope(loginID, 5))
		case "/v2/cli.exchangeToken":
			var body components.V2CliExchangeTokenRequestBody1
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Equal(t, loginID, body.LoginID)
			require.Nil(t, body.RefreshToken)
			mu.Lock()
			polls++
			n := polls
			mu.Unlock()
			switch n {
			case 1:
				w.Header().Set("Retry-After", "1")
				writeEnvelope(t, w, http.StatusAccepted, pendingEnvelope())
			case 2, 3:
				w.Header().Set("Retry-After", "2")
				writeEnvelope(t, w, http.StatusAccepted, pendingEnvelope())
			default:
				writeEnvelope(t, w, http.StatusOK, completeEnvelope(workspaceID, sessionID))
			}
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	var slept []time.Duration
	opened := ""
	err := run(context.Background(), options{
		apiURL:    srv.URL,
		noBrowser: false,
		keys:      store,
		client:    srv.Client(),
		sleep: func(_ context.Context, d time.Duration) error {
			slept = append(slept, d)
			return nil
		},
		open: func(url string) error {
			opened = url
			return nil
		},
	})
	require.NoError(t, err)
	require.NotNil(t, startedBody.DeviceName)
	require.NotEmpty(t, *startedBody.DeviceName)
	require.Equal(t, verificationURL, opened)
	require.Equal(t, []time.Duration{time.Second, 2 * time.Second, 2 * time.Second}, slept)

	stored, err := session.Active()
	require.NoError(t, err)
	require.Equal(t, "access_issued", stored.AccessToken)
	require.Equal(t, "refresh_issued", stored.RefreshToken)
	require.Equal(t, workspaceID, stored.WorkspaceID)
	require.Equal(t, sessionID, stored.SessionID)
	require.Equal(t, srv.URL, stored.APIURL)
}

func TestDeviceLoginStopsWhenDenied(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/cli.startDeviceLogin" {
			writeEnvelope(t, w, http.StatusOK, startedEnvelope(uid.New(uid.DeviceAuthorizationPrefix), 5))
			return
		}
		writeEnvelope(t, w, http.StatusForbidden, apierrors.ForbiddenErrorResponse{
			Error_: components.BaseError{Detail: "Authorization was denied.", Status: http.StatusForbidden},
			Meta:   components.Meta{RequestID: uid.New(uid.RequestPrefix)},
		})
	}))
	t.Cleanup(srv.Close)

	err := run(context.Background(), options{
		apiURL:    srv.URL,
		noBrowser: true,
		client:    srv.Client(),
		sleep: func(context.Context, time.Duration) error {
			t.Fatal("denied login must not keep polling")
			return nil
		},
	})
	var response *sdk.ResponseError
	require.ErrorAs(t, err, &response)
	require.Equal(t, http.StatusForbidden, response.StatusCode)
	require.EqualError(t, err, "Authorization was denied.")
}

func TestDeviceLoginRetriesTransientPollErrors(t *testing.T) {
	useSessionStore(t)
	t.Setenv("HOME", t.TempDir())
	store := &memoryRootKeyStore{}
	polls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/cli.startDeviceLogin" {
			writeEnvelope(t, w, http.StatusOK, startedEnvelope(uid.New(uid.DeviceAuthorizationPrefix), 1))
			return
		}
		polls++
		if polls == 1 {
			writeEnvelope(t, w, http.StatusServiceUnavailable, apierrors.InternalServerErrorResponse{
				Error_: components.BaseError{Detail: "try again", Status: http.StatusServiceUnavailable},
				Meta:   components.Meta{RequestID: uid.New(uid.RequestPrefix)},
			})
			return
		}
		writeEnvelope(t, w, http.StatusOK, completeEnvelope(uid.New(uid.WorkspacePrefix), uid.New(uid.CLISessionPrefix)))
	}))
	t.Cleanup(srv.Close)

	err := run(context.Background(), options{
		apiURL:    srv.URL,
		noBrowser: true,
		keys:      store,
		client:    srv.Client(),
		sleep:     func(context.Context, time.Duration) error { return nil },
	})
	require.NoError(t, err)
	require.Equal(t, 2, polls)
}

func TestDeviceLoginStopsOnClientPollErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/cli.startDeviceLogin" {
			writeEnvelope(t, w, http.StatusOK, startedEnvelope(uid.New(uid.DeviceAuthorizationPrefix), 1))
			return
		}
		writeEnvelope(t, w, http.StatusBadRequest, apierrors.BadRequestErrorResponse{
			Error_: components.BadRequestErrorDetails{
				Detail: "This login was already used.",
				Status: http.StatusBadRequest,
				Errors: []components.ValidationError{},
			},
			Meta: components.Meta{RequestID: uid.New(uid.RequestPrefix)},
		})
	}))
	t.Cleanup(srv.Close)

	err := run(context.Background(), options{
		apiURL:    srv.URL,
		noBrowser: true,
		client:    srv.Client(),
		sleep: func(context.Context, time.Duration) error {
			t.Fatal("client errors must not be retried")
			return nil
		},
	})
	require.EqualError(t, err, "This login was already used.")
}

func TestDeviceLoginRevokesTheSessionWhenItCannotBeStored(t *testing.T) {
	store := useSessionStore(t)
	store.SetError(errors.New("keychain rejected the blob"))
	home := filepath.Join(t.TempDir(), "not-a-directory")
	require.NoError(t, os.WriteFile(home, []byte("x"), 0o600))
	t.Setenv("HOME", home)
	var revoked bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/cli.startDeviceLogin" {
			writeEnvelope(t, w, http.StatusOK, startedEnvelope(uid.New(uid.DeviceAuthorizationPrefix), 1))
			return
		}
		if r.URL.Path == "/v2/cli.revokeSession" {
			revoked = r.Header.Get("authorization") == "Bearer access_issued"
			writeEnvelope(t, w, http.StatusOK, components.V2CliRevokeSessionResponseBody{
				Data: components.EmptyResponse{},
				Meta: components.Meta{RequestID: uid.New(uid.RequestPrefix)},
			})
			return
		}
		writeEnvelope(t, w, http.StatusOK, completeEnvelope(uid.New(uid.WorkspacePrefix), uid.New(uid.CLISessionPrefix)))
	}))
	t.Cleanup(srv.Close)

	stderr := captureStderr(t, func() {
		err := run(context.Background(), options{
			apiURL:    srv.URL,
			noBrowser: true,
			client:    srv.Client(),
			sleep:     func(context.Context, time.Duration) error { return nil },
		})
		require.ErrorContains(t, err, "could not store the CLI session, so it was revoked")
	})
	require.True(t, revoked)
	require.NotContains(t, stderr, "refresh_issued")
}

func TestPollUntilKey(t *testing.T) {
	complete := completeEnvelope(uid.New(uid.WorkspacePrefix), uid.New(uid.CLISessionPrefix))
	withoutTokens := completeEnvelope(uid.New(uid.WorkspacePrefix), uid.New(uid.CLISessionPrefix))
	withoutTokens.Data.AccessToken = ""
	withoutTokens.Data.RefreshToken = ""

	htmlBadGateway := func(w http.ResponseWriter) {
		w.Header().Set("content-type", "text/html")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("<html><body>502 Bad Gateway</body></html>"))
	}

	cases := []struct {
		name      string
		responses []func(w http.ResponseWriter)
		deadline  time.Time
		wantErr   string
		wantPolls int
	}{
		{
			name:      "deadline passed",
			deadline:  time.Now().Add(-time.Second),
			wantErr:   "login timed out. Run unkey login again",
			wantPolls: 0,
		},
		{
			name: "expired token",
			responses: []func(w http.ResponseWriter){
				func(w http.ResponseWriter) {
					writeEnvelope(t, w, http.StatusGone, apierrors.GoneErrorResponse{
						Error_: components.BaseError{Detail: "This login expired.", Status: http.StatusGone},
						Meta:   components.Meta{RequestID: uid.New(uid.RequestPrefix)},
					})
				},
			},
			wantErr:   "This login expired.",
			wantPolls: 1,
		},
		{
			name: "complete without tokens",
			responses: []func(w http.ResponseWriter){
				func(w http.ResponseWriter) { writeEnvelope(t, w, http.StatusOK, withoutTokens) },
			},
			wantErr:   "login completed without a session",
			wantPolls: 1,
		},
		{
			name: "rate limited then complete",
			responses: []func(w http.ResponseWriter){
				func(w http.ResponseWriter) {
					writeEnvelope(t, w, http.StatusTooManyRequests, apierrors.TooManyRequestsErrorResponse{
						Error_: components.BaseError{Detail: "slow down", Status: http.StatusTooManyRequests},
						Meta:   components.Meta{RequestID: uid.New(uid.RequestPrefix)},
					})
				},
				func(w http.ResponseWriter) { writeEnvelope(t, w, http.StatusOK, complete) },
			},
			wantPolls: 2,
		},
		{
			name: "html bad gateway then complete",
			responses: []func(w http.ResponseWriter){
				htmlBadGateway,
				func(w http.ResponseWriter) { writeEnvelope(t, w, http.StatusOK, complete) },
			},
			wantPolls: 2,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var mu sync.Mutex
			polls := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				mu.Lock()
				n := polls
				polls++
				mu.Unlock()
				require.Less(t, n, len(tc.responses))
				tc.responses[n](w)
			}))
			t.Cleanup(srv.Close)

			issued, err := pollUntilKey(context.Background(), srv.Client(), srv.URL, uid.New(uid.DeviceAuthorizationPrefix), tc.deadline, time.Second, func(context.Context, time.Duration) error { return nil })
			mu.Lock()
			require.Equal(t, tc.wantPolls, polls)
			mu.Unlock()
			if tc.wantErr != "" {
				require.EqualError(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, complete.Data.AccessToken, issued.AccessToken)
			require.Equal(t, complete.Data.WorkspaceID, issued.WorkspaceID)
		})
	}
}

func TestPollUntilKeyRejectsIncompleteSession(t *testing.T) {
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
			response := completeEnvelope(uid.New(uid.WorkspacePrefix), uid.New(uid.CLISessionPrefix))
			tc.change(&response.Data)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				writeEnvelope(t, w, http.StatusOK, response)
			}))
			t.Cleanup(srv.Close)

			issued, err := pollUntilKey(t.Context(), srv.Client(), srv.URL, uid.New(uid.DeviceAuthorizationPrefix), time.Time{}, time.Second, nil)
			require.EqualError(t, err, "login completed without a session")
			require.Empty(t, issued)
		})
	}
}

const verificationURL = "http://dashboard.example/cli/device?user_code=ABCD-EFGH"

func writeEnvelope[T any](t *testing.T, w http.ResponseWriter, status int, envelope T) {
	t.Helper()
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	require.NoError(t, json.NewEncoder(w).Encode(envelope))
}

func startedEnvelope(loginID string, interval int64) components.V2CliStartDeviceLoginResponseBody {
	return components.V2CliStartDeviceLoginResponseBody{
		Data: components.V2CliStartDeviceLoginResponseData{
			LoginID:         loginID,
			UserCode:        "ABCD-EFGH",
			VerificationURL: verificationURL,
			ExpiresIn:       300,
			Interval:        interval,
		},
		Meta: components.Meta{RequestID: uid.New(uid.RequestPrefix)},
	}
}

func pendingEnvelope() components.V2CliExchangeTokenPendingResponseBody {
	return components.V2CliExchangeTokenPendingResponseBody{
		Data: components.EmptyResponse{},
		Meta: components.Meta{RequestID: uid.New(uid.RequestPrefix)},
	}
}

func completeEnvelope(workspaceID string, sessionID string) components.V2CliExchangeTokenResponseBody {
	return components.V2CliExchangeTokenResponseBody{
		Data: components.V2CliExchangeTokenResponseData{
			AccessToken:  "access_issued",
			RefreshToken: "refresh_issued",
			ExpiresIn:    3600,
			WorkspaceID:  workspaceID,
			SessionID:    sessionID,
		},
		Meta: components.Meta{RequestID: uid.New(uid.RequestPrefix)},
	}
}

func TestReadRootKeyFromStdin(t *testing.T) {
	key, err := readRootKeyFrom(strings.NewReader("  unkey_piped\nignored\n"))
	require.NoError(t, err)
	require.Equal(t, "unkey_piped", key)

	_, err = readRootKeyFrom(strings.NewReader("\n"))
	require.EqualError(t, err, "no root key on stdin")
}

type memoryRootKeyStore struct {
	secret      string
	unavailable bool
	setErr      error
}

var _ credentials.RootKeyStore = (*memoryRootKeyStore)(nil)

func (m *memoryRootKeyStore) Get() (string, error) {
	if m.unavailable {
		return "", keystore.ErrUnavailable
	}
	if m.secret == "" {
		return "", keystore.ErrNotFound
	}
	return m.secret, nil
}

func (m *memoryRootKeyStore) Set(secret string) error {
	if m.setErr != nil {
		return m.setErr
	}
	if m.unavailable {
		return keystore.ErrUnavailable
	}
	m.secret = secret
	return nil
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	original := os.Stdout
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = original })
	fn()
	os.Stdout = original
	require.NoError(t, w.Close())
	out, err := io.ReadAll(r)
	require.NoError(t, err)
	return string(out)
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	original := os.Stderr
	os.Stderr = w
	t.Cleanup(func() { os.Stderr = original })
	fn()
	os.Stderr = original
	require.NoError(t, w.Close())
	out, err := io.ReadAll(r)
	require.NoError(t, err)
	return string(out)
}

func configBody(t *testing.T, home string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(home, ".unkey", "config.toml"))
	if os.IsNotExist(err) {
		return ""
	}
	require.NoError(t, err)
	return string(body)
}
