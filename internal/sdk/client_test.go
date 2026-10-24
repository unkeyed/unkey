package sdk_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/internal/sdk"
	"github.com/unkeyed/unkey/internal/sdk/models/apierrors"
	"github.com/unkeyed/unkey/internal/sdk/models/components"
	"github.com/unkeyed/unkey/internal/sdk/models/operations"
	"github.com/unkeyed/unkey/internal/sdk/polling"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestClientUsesTypedBodiesAndRequestScopedCredentials(t *testing.T) {
	t.Setenv("UNKEY_ROOT_KEY", "unrelated-root-key")
	sessionID := uid.New(uid.CLISessionPrefix)
	started := components.V2CliStartDeviceLoginResponseBody{
		Data: components.V2CliStartDeviceLoginResponseData{
			LoginID:                 uid.New(uid.DeviceAuthorizationPrefix),
			UserCode:                "BCDF-GHJK",
			VerificationURIComplete: "https://app.unkey.com/cli/device?user_code=BCDF-GHJK",
			ExpiresIn:               600,
			Interval:                5,
		},
		Meta: components.Meta{RequestID: uid.New(uid.RequestPrefix)},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "application/json", r.Header.Get("Accept"))
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/proxy/v2/cli.revokeSession":
			require.Equal(t, "Bearer access-token", r.Header.Get("Authorization"))
			var body components.V2CliRevokeSessionRequestBody
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Equal(t, sessionID, body.SessionID)
			require.NoError(t, json.NewEncoder(w).Encode(components.V2CliRevokeSessionResponseBody{
				Data: components.EmptyResponse{},
				Meta: components.Meta{RequestID: uid.New(uid.RequestPrefix)},
			}))
		case "/proxy/v2/cli.startDeviceLogin":
			require.Empty(t, r.Header.Get("Authorization"))
			var body map[string]json.RawMessage
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.NotContains(t, body, "deviceName")
			require.NoError(t, json.NewEncoder(w).Encode(started))
		default:
			t.Errorf("unexpected request path %q", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	client := sdk.NewClient(server.URL+"/proxy", server.Client())

	revoked, err := client.Cli.RevokeSession(t.Context(), components.V2CliRevokeSessionRequestBody{SessionID: sessionID}, sdk.Bearer("access-token"))
	require.NoError(t, err)
	require.NotNil(t, revoked.V2CliRevokeSessionResponseBody)
	res, err := client.Cli.StartDeviceLogin(t.Context(), components.V2CliStartDeviceLoginRequestBody{})
	require.NoError(t, err)
	require.Equal(t, &started, res.V2CliStartDeviceLoginResponseBody)
}

func TestListSessionsSendsNoBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v2/cli.listSessions", r.URL.Path)
		require.Equal(t, "Bearer access-token", r.Header.Get("Authorization"))
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.Empty(t, body)
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(components.V2CliListSessionsResponseBody{
			Data: components.V2CliListSessionsResponseData{Sessions: []components.V2CliListSessionsSession{}},
			Meta: components.Meta{RequestID: uid.New(uid.RequestPrefix)},
		}))
	}))
	t.Cleanup(server.Close)

	res, err := sdk.NewClient(server.URL, server.Client()).Cli.ListSessions(t.Context(), sdk.Bearer("access-token"))
	require.NoError(t, err)
	require.NotNil(t, res.V2CliListSessionsResponseBody)
	require.Empty(t, res.V2CliListSessionsResponseBody.Data.Sessions)
}

func TestDecideDeviceLoginSerializesApproveAndDeny(t *testing.T) {
	for _, tc := range []struct {
		decision string
		request  components.V2CliDecideDeviceLoginRequestBody
	}{
		{
			decision: "approve",
			request: components.CreateV2CliDecideDeviceLoginRequestBodyOne(components.One{
				Decision: components.DecisionApprove, UserCode: "BCDF-GHJK", Permissions: []string{"api.*.read_api"},
			}),
		},
		{
			decision: "deny",
			request: components.CreateV2CliDecideDeviceLoginRequestBodyTwo(components.Two{
				Decision: components.V2CliDecideDeviceLoginRequestBodyDecisionDeny, UserCode: "BCDF-GHJK",
			}),
		},
	} {
		t.Run(tc.decision, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, http.MethodPost, r.Method)
				require.Equal(t, "/v2/cli.decideDeviceLogin", r.URL.Path)
				var body components.Two
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				require.Equal(t, tc.decision, string(body.Decision))
				require.Equal(t, "BCDF-GHJK", body.UserCode)
				if tc.decision == "approve" {
					require.Equal(t, []string{"api.*.read_api"}, body.Permissions)
				} else {
					require.Nil(t, body.Permissions)
				}
				w.Header().Set("Content-Type", "application/json")
				require.NoError(t, json.NewEncoder(w).Encode(components.V2CliDecideDeviceLoginResponseBody{
					Data: components.EmptyResponse{},
					Meta: components.Meta{RequestID: uid.New(uid.RequestPrefix)},
				}))
			}))
			t.Cleanup(server.Close)

			res, err := sdk.NewClient(server.URL, server.Client()).Cli.DecideDeviceLogin(t.Context(), tc.request)
			require.NoError(t, err)
			require.NotNil(t, res.V2CliDecideDeviceLoginResponseBody)
		})
	}
}

func TestClientPreservesHTTPErrorsWithoutRetrying(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   any
		detail string
	}{
		{name: "validation", status: 400, body: apierrors.BadRequestErrorResponse{Error_: components.BadRequestErrorDetails{Detail: "refreshToken is required"}}, detail: "refreshToken is required"},
		{name: "unauthorized", status: 401, body: apierrors.UnauthorizedErrorResponse{Error_: components.BaseError{Detail: "session revoked"}}, detail: "session revoked"},
		{name: "title fallback", status: 403, body: apierrors.ForbiddenErrorResponse{Error_: components.BaseError{Title: "Forbidden"}}, detail: "Forbidden"},
		{name: "rate limited", status: 429, body: apierrors.TooManyRequestsErrorResponse{Error_: components.BaseError{Detail: "try later"}}, detail: "try later"},
		{name: "server failure", status: 500, body: apierrors.InternalServerErrorResponse{Error_: components.BaseError{Detail: "try again"}}, detail: "try again"},
		{name: "non-JSON gateway failure", status: 502, body: nil, detail: "unkey API returned 502"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls.Add(1)
				w.WriteHeader(tc.status)
				if tc.body == nil {
					_, err := io.WriteString(w, "<html>bad gateway</html>")
					require.NoError(t, err)
					return
				}
				require.NoError(t, json.NewEncoder(w).Encode(tc.body))
			}))
			t.Cleanup(server.Close)
			client := sdk.NewClient(server.URL, nil)
			request := components.CreateV2CliExchangeTokenRequestBodyV2CliExchangeTokenRequestBody2(components.V2CliExchangeTokenRequestBody2{RefreshToken: "refresh-token"})
			_, err := client.Cli.ExchangeToken(t.Context(), request)
			var response *sdk.ResponseError
			require.ErrorAs(t, err, &response)
			require.Equal(t, tc.status, response.StatusCode)
			require.EqualError(t, response, tc.detail)
			require.Equal(t, int32(1), calls.Load())
		})
	}
}

func TestClientBoundsResponseSize(t *testing.T) {
	for _, size := range []int{1 << 20, 1<<20 + 1} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			body, err := json.Marshal(components.V2CliRevokeSessionResponseBody{
				Data: components.EmptyResponse{},
				Meta: components.Meta{RequestID: uid.New(uid.RequestPrefix)},
			})
			require.NoError(t, err)
			payload := string(body) + strings.Repeat(" ", size-len(body))
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, err := io.WriteString(w, payload)
				require.NoError(t, err)
			}))
			t.Cleanup(server.Close)
			client := sdk.NewClient(server.URL, nil)
			res, err := client.Cli.RevokeSession(t.Context(), components.V2CliRevokeSessionRequestBody{SessionID: uid.New(uid.CLISessionPrefix)})
			if size > 1<<20 {
				var response *sdk.ResponseError
				require.ErrorAs(t, err, &response)
				require.EqualError(t, response, "unkey API response exceeds 1 MiB")
				return
			}
			require.NoError(t, err)
			require.NotNil(t, res.V2CliRevokeSessionResponseBody)
		})
	}
}

func TestClientPreservesTransferAndCancellationErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "100")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)
	client := sdk.NewClient(server.URL, nil)
	_, err := client.Cli.StartDeviceLogin(t.Context(), components.V2CliStartDeviceLoginRequestBody{})
	var transport *sdk.TransportError
	require.ErrorAs(t, err, &transport)
	require.ErrorIs(t, err, io.ErrUnexpectedEOF)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = client.Cli.StartDeviceLogin(ctx, components.V2CliStartDeviceLoginRequestBody{})
	require.ErrorAs(t, err, &transport)
	require.ErrorIs(t, err, context.Canceled)
}

func TestGeneratedDeviceLoginPollingStopsAtTerminalStatusOrLimit(t *testing.T) {
	for _, status := range []components.Status{components.StatusComplete, components.StatusAccessDenied, components.StatusExpiredToken, components.StatusPermissionsRequired} {
		t.Run(string(status), func(t *testing.T) {
			loginID := uid.New(uid.DeviceAuthorizationPrefix)
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/v2/cli.exchangeToken", r.URL.Path)
				var body components.V2CliExchangeTokenRequestBody1
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				require.Equal(t, loginID, body.LoginID)
				require.Nil(t, body.RefreshToken)
				responseStatus := components.StatusPermissionsRequired
				if calls.Add(1) > 1 {
					responseStatus = status
				}
				w.Header().Set("Content-Type", "application/json")
				require.NoError(t, json.NewEncoder(w).Encode(components.V2CliExchangeTokenResponseBody{
					Data: components.V2CliExchangeTokenResponseData{Status: responseStatus, Interval: 5},
					Meta: components.Meta{RequestID: uid.New(uid.RequestPrefix)},
				}))
			}))
			t.Cleanup(server.Close)
			client := sdk.NewClient(server.URL, server.Client())
			request := components.CreateV2CliExchangeTokenRequestBodyV2CliExchangeTokenRequestBody1(components.V2CliExchangeTokenRequestBody1{LoginID: loginID})
			res, err := client.Cli.ExchangeToken(t.Context(), request, operations.WithPolling(
				client.Cli.ExchangeTokenWaitForCompletion(), polling.WithIntervalSecondsOverride(0), polling.WithLimitCountOverride(2),
			))
			switch status {
			case components.StatusComplete:
				require.NoError(t, err)
			case components.StatusAccessDenied, components.StatusExpiredToken:
				var terminal *polling.FailureCriteriaError
				require.ErrorAs(t, err, &terminal)
			case components.StatusPermissionsRequired:
				var limit *polling.LimitCountError
				require.ErrorAs(t, err, &limit)
			}
			require.NotNil(t, res)
			require.Equal(t, status, res.V2CliExchangeTokenResponseBody.Data.Status)
			require.Equal(t, int32(2), calls.Load())
		})
	}
}

func TestDeviceLoginWaitHonorsDeadlineDuringServerIntervalAndRetries(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusTooManyRequests, http.StatusServiceUnavailable} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls.Add(1)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				require.NoError(t, json.NewEncoder(w).Encode(components.V2CliExchangeTokenResponseBody{
					Data: components.V2CliExchangeTokenResponseData{Status: components.StatusPermissionsRequired, Interval: 3600},
					Meta: components.Meta{RequestID: uid.New(uid.RequestPrefix)},
				}))
			}))
			t.Cleanup(server.Close)
			ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
			t.Cleanup(cancel)
			result := make(chan error, 1)
			go func() {
				request := components.CreateV2CliExchangeTokenRequestBodyV2CliExchangeTokenRequestBody1(components.V2CliExchangeTokenRequestBody1{LoginID: uid.New(uid.DeviceAuthorizationPrefix)})
				_, err := sdk.NewClient(server.URL, server.Client()).WaitForDeviceLogin(ctx, request, time.Hour, nil)
				result <- err
			}()
			select {
			case err := <-result:
				require.ErrorIs(t, err, context.DeadlineExceeded)
			case <-time.After(2 * time.Second):
				t.Fatal("polling did not stop at its deadline")
			}
			require.Equal(t, int32(1), calls.Load())
		})
	}
}
