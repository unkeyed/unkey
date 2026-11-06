package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/internal/services/keys/metrics"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/logger/loggertest"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_root_keys_create_key"
)

func TestRootKeyAuthenticationRestrictsNetworks(t *testing.T) {
	h, route, p := newHarness(t)
	var events []auditlog.Event
	auditLogBuffer := batch.New(batch.Config[auditlog.Event]{
		Name:          "ip_denial_test",
		BatchSize:     1,
		BufferSize:    10,
		FlushInterval: time.Hour,
		Consumers:     1,
		Drop:          false,
		Flush:         func(_ context.Context, batch []auditlog.Event) { events = append(events, batch...) },
	})
	t.Cleanup(auditLogBuffer.Close)
	keyConfig := keys.Config{
		DB:             db.ToMySQL(h.DB),
		KeyCache:       h.Caches.VerificationKeyByHash,
		RootKeyCache:   h.Caches.RootKeyByHash,
		RateLimiter:    h.Ratelimit,
		RBAC:           rbac.New(),
		Region:         "test",
		UsageLimiter:   h.UsageLimiter,
		Source:         "api",
		AuditLogBuffer: nil,
	}
	keyService, err := keys.New(keyConfig)
	require.ErrorContains(t, err, "audit log buffer is required")
	require.Nil(t, keyService)
	keyConfig.AuditLogBuffer = auditLogBuffer
	keyService, err = keys.New(keyConfig)
	require.NoError(t, err)
	metaExpected := []map[string]any{}
	ipsExpected := []string{}
	startedAt := time.Now().UnixMilli()
	logs := loggertest.Install(t)
	logger.SetSampler(logger.TailSampler{SampleRate: 0, SlowThreshold: time.Hour})
	t.Cleanup(func() { logger.SetSampler(logger.AlwaysSample{}) })
	denied := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
		"Authorization": {"Bearer test"}, "Content-Type": {"application/json"},
	}, handler.Request{Permissions: []string{}, IpAllowlist: &[]string{"198.51.100.128/25"}})
	require.Equal(t, http.StatusForbidden, denied.Status, "%s", denied.RawBody)
	require.NoError(t, db.Query.UpdateIPAllowlistEnabled(t.Context(), h.DB.RW(), db.UpdateIPAllowlistEnabledParams{WorkspaceID: p.AuthorizedWorkspaceID, Enabled: true}))
	h.Caches.WorkspaceLimits.Remove(t.Context(), p.AuthorizedWorkspaceID)
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
		"Authorization": {"Bearer test"}, "Content-Type": {"application/json"},
	}, handler.Request{Permissions: []string{}, IpAllowlist: &[]string{"198.51.100.128/25", "2001:db8:1234::/48"}})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
	require.NoError(t, db.Query.UpdateIPAllowlistEnabled(t.Context(), h.DB.RW(), db.UpdateIPAllowlistEnabledParams{WorkspaceID: p.AuthorizedWorkspaceID, Enabled: false}))
	// Disablement takes effect after the shared limits cache refreshes.
	cached := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
		"Authorization": {"Bearer test"}, "Content-Type": {"application/json"},
	}, handler.Request{Permissions: []string{}, IpAllowlist: &[]string{"198.51.100.128/25"}})
	require.Equal(t, http.StatusOK, cached.Status, "%s", cached.RawBody)
	h.Caches.WorkspaceLimits.Remove(t.Context(), p.AuthorizedWorkspaceID)
	denied = testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
		"Authorization": {"Bearer test"}, "Content-Type": {"application/json"},
	}, handler.Request{Permissions: []string{}, IpAllowlist: &[]string{"198.51.100.128/25"}})
	require.Equal(t, http.StatusForbidden, denied.Status, "%s", denied.RawBody)
	for _, test := range []struct {
		address string
		allowed bool
	}{
		{"198.51.100.127:1234", false},
		{"198.51.100.128:1234", true},
		{"198.51.100.255:1234", true},
		{"198.51.101.1:1234", false},
		{"[2001:db8:1234::42]:1234", true},
		{"[2001:db8:1235::42]:1234", false},
		{"", false},
	} {
		t.Run(test.address, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			req.RemoteAddr = test.address
			req.Header.Set("Authorization", "Bearer "+res.Body.Data.Key)
			req.Header.Set("X-Forwarded-For", "198.51.100.128")
			req.Header.Set("User-Agent", "ip-denial-test")
			session := &zen.Session{}
			require.NoError(t, session.Init(httptest.NewRecorder(), req, 0))
			validBefore := rootKeyVerificationCount(t, "VALID", "none")
			notFoundBefore := rootKeyVerificationCount(t, "NOT_FOUND", "ip_denied")
			var denialsBefore, denialsAfter dto.Metric
			counter, counterErr := metrics.AuthIPAllowlistDenialsTotal.GetMetricWithLabelValues()
			require.NoError(t, counterErr)
			require.NoError(t, counter.Write(&denialsBefore))
			ctx, event := logger.StartWideEvent(t.Context(), "root key authentication")
			snapshot := logs.Snapshot()
			_, err := keyService.GetRootKey(ctx, session)
			event.End()
			warnings := []slog.Record{}
			for _, record := range logs.Since(snapshot) {
				if record.Message == "This IP address is not allowed to use the root key." {
					warnings = append(warnings, record)
				}
			}
			require.NoError(t, counter.Write(&denialsAfter))
			if test.allowed {
				require.Empty(t, warnings)
				require.NoError(t, err)
				require.Equal(t, denialsBefore.GetCounter().GetValue(), denialsAfter.GetCounter().GetValue())
				require.Equal(t, validBefore+1, rootKeyVerificationCount(t, "VALID", "none"))
				require.Equal(t, notFoundBefore, rootKeyVerificationCount(t, "NOT_FOUND", "ip_denied"))
			} else {
				ipsExpected = append(ipsExpected, session.ClientIP().String())
				metaExpected = append(metaExpected, map[string]any{
					"request_id":  session.RequestID(),
					"allowed_ips": []string{"198.51.100.128/25", "2001:db8:1234::/48"},
				})
				require.Empty(t, warnings)
				require.Error(t, err)
				require.Equal(t, denialsBefore.GetCounter().GetValue()+1, denialsAfter.GetCounter().GetValue())
				code, ok := fault.GetCode(err)
				require.True(t, ok)
				require.Equal(t, codes.Auth.Authentication.KeyNotFound.URN(), code)
				require.Equal(t, "The provided root key is invalid.", fault.UserFacingMessage(err))
				require.Contains(t, fault.InternalMessage(err), session.ClientIP().String())
				require.Contains(t, fault.InternalMessage(err), "This IP address is not allowed to use the root key.")
				require.Equal(t, notFoundBefore+1, rootKeyVerificationCount(t, "NOT_FOUND", "ip_denied"))
				require.Equal(t, validBefore, rootKeyVerificationCount(t, "VALID", "none"))
			}
		})
	}
	auditLogBuffer.Close()
	require.Len(t, events, len(metaExpected))
	for i, event := range events {
		require.Equal(t, "auth.ipDenied", event.Event)
		require.Equal(t, p.AuthorizedWorkspaceID, event.WorkspaceID)
		require.Equal(t, res.Body.Data.KeyId, event.Actor.ID)
		require.Len(t, event.Targets, 1)
		require.Equal(t, res.Body.Data.KeyId, event.Targets[0].ID)
		require.Equal(t, ipsExpected[i], event.RemoteIP)
		require.Equal(t, "ip-denial-test", event.UserAgent)
		require.Equal(t, fmt.Sprintf("Root key authentication from %s denied by IP allowlist", ipsExpected[i]), event.Description)
		require.GreaterOrEqual(t, event.Time, startedAt)
		require.LessOrEqual(t, event.Time, time.Now().UnixMilli())
		require.Equal(t, metaExpected[i], event.Meta)
	}
}

func rootKeyVerificationCount(t *testing.T, code, reason string) float64 {
	t.Helper()
	var metric dto.Metric
	require.NoError(t, metrics.KeyVerificationsTotal.WithLabelValues("root_key", code, reason).Write(&metric))
	return metric.GetCounter().GetValue()
}

// TestRootKeyAuthenticationPrefersNewStore guarantees migration overlap resolves
// to the new root-key row before the legacy fallback is queried.
func TestRootKeyAuthenticationPrefersNewStore(t *testing.T) {
	h := testutil.NewHarness(t)
	r := h.Resources()
	legacy := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID:    r.RootWorkspace.ID,
		KeySpaceID:     r.RootKeySpace.ID,
		ForWorkspaceID: &r.UserWorkspace.ID,
	})
	newID := uid.New(uid.KeyPrefix)
	require.NoError(t, db.Query.InsertUnkeyRootKey(t.Context(), h.DB.RW(), db.InsertUnkeyRootKeyParams{
		ID:          newID,
		WorkspaceID: r.UserWorkspace.ID,
		Hash:        hash.Sha256(legacy.Key),
		Name:        sql.NullString{},
		Prefix:      "unkey",
		Start:       "test",
		End:         "test",
		Enabled:     true,
		Expires:     sql.NullInt64{},
		CreatedAt:   1_700_000_000_000,
	}))

	request := httptest.NewRequest(http.MethodPost, "/", nil)
	request.Header.Set("Authorization", "Bearer "+legacy.Key)
	session := &zen.Session{}
	require.NoError(t, session.Init(httptest.NewRecorder(), request, 0))

	rootKey, err := h.Keys.GetRootKey(t.Context(), session)
	require.NoError(t, err)
	require.Equal(t, newID, rootKey.Key.ID)

	_, err = db.Query.SoftDeleteUnkeyRootKey(t.Context(), h.DB.RW(), db.SoftDeleteUnkeyRootKeyParams{
		Now:         sql.NullInt64{Int64: 1, Valid: true},
		ID:          newID,
		WorkspaceID: r.UserWorkspace.ID,
	})
	require.NoError(t, err)
	h.Caches.RootKeyByHash.Remove(t.Context(), hash.Sha256(legacy.Key))
	for range 2 {
		_, err = h.Keys.GetRootKey(t.Context(), session)
		require.Error(t, err)
	}
}

// TestNewRootKeyAuthenticationIgnoresLegacyOwnership guarantees new root keys
// do not depend on the internal workspace, API, or keyspace used by legacy keys.
func TestNewRootKeyAuthenticationIgnoresLegacyOwnership(t *testing.T) {
	h, route, p := newHarness(t)
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
		"Authorization": {"Bearer test"}, "Content-Type": {"application/json"},
	}, handler.Request{Permissions: []string{}})
	require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)

	r := h.Resources()
	_, err := db.Query.UpdateWorkspaceEnabled(t.Context(), h.DB.RW(), db.UpdateWorkspaceEnabledParams{Enabled: false, ID: r.RootWorkspace.ID})
	require.NoError(t, err)
	require.NoError(t, db.Query.SoftDeleteApi(t.Context(), h.DB.RW(), db.SoftDeleteApiParams{
		Now:   sql.NullInt64{Int64: 1, Valid: true},
		ApiID: r.RootApi.ID,
	}))
	_, err = h.DB.RW().ExecContext(t.Context(), "DELETE FROM key_auth WHERE id = ?", r.RootKeySpace.ID)
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodPost, "/", nil)
	request.Header.Set("Authorization", "Bearer "+res.Body.Data.Key)
	session := &zen.Session{}
	require.NoError(t, session.Init(httptest.NewRecorder(), request, 0))
	rootKey, err := h.Keys.GetRootKey(t.Context(), session)
	require.NoError(t, err)
	require.Equal(t, p.AuthorizedWorkspaceID, rootKey.AuthorizedWorkspaceID)
}

// TestNewRootKeyAuthenticationChecksLifecycle guarantees the new storage does not
// bypass lifecycle checks. For example, expired or disabled keys fail auth, as do
// keys whose target workspace is missing or disabled.
func TestNewRootKeyAuthenticationChecksLifecycle(t *testing.T) {
	for _, tt := range []struct {
		name   string
		mutate func(t *testing.T, h *testutil.Harness, keyID, workspaceID string)
		code   codes.URN
	}{
		{"disabled", func(t *testing.T, h *testutil.Harness, keyID, workspaceID string) {
			require.NoError(t, db.Query.UpdateUnkeyRootKey(t.Context(), h.DB.RW(), db.UpdateUnkeyRootKeyParams{
				EnabledSpecified: 1,
				Enabled:          sql.NullBool{Bool: false, Valid: true},
				ID:               keyID,
				WorkspaceID:      workspaceID,
			}))
		}, codes.Auth.Authorization.KeyDisabled.URN()},
		{"expired", func(t *testing.T, h *testutil.Harness, keyID, workspaceID string) {
			require.NoError(t, db.Query.UpdateUnkeyRootKeyExpiration(t.Context(), h.DB.RW(), db.UpdateUnkeyRootKeyExpirationParams{
				Expires:     sql.NullInt64{Int64: 946684800000, Valid: true},
				ID:          keyID,
				WorkspaceID: workspaceID,
			}))
		}, codes.Auth.Authorization.Forbidden.URN()},
		{"deleted", func(t *testing.T, h *testutil.Harness, keyID, workspaceID string) {
			_, err := db.Query.SoftDeleteUnkeyRootKey(t.Context(), h.DB.RW(), db.SoftDeleteUnkeyRootKeyParams{
				Now:         sql.NullInt64{Int64: 1, Valid: true},
				ID:          keyID,
				WorkspaceID: workspaceID,
			})
			require.NoError(t, err)
		}, codes.Auth.Authentication.KeyNotFound.URN()},
		{"missing target", func(t *testing.T, h *testutil.Harness, keyID, _ string) {
			_, err := h.DB.RW().ExecContext(t.Context(), "UPDATE unkey_root_keys SET workspace_id = ? WHERE id = ?", uid.New(uid.WorkspacePrefix), keyID)
			require.NoError(t, err)
		}, codes.Data.Workspace.NotFound.URN()},
		{"disabled target", func(t *testing.T, h *testutil.Harness, _, workspaceID string) {
			_, err := db.Query.UpdateWorkspaceEnabled(t.Context(), h.DB.RW(), db.UpdateWorkspaceEnabledParams{Enabled: false, ID: workspaceID})
			require.NoError(t, err)
		}, codes.Auth.Authorization.WorkspaceDisabled.URN()},
	} {
		t.Run(tt.name, func(t *testing.T) {
			h, route, p := newHarness(t)
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
				"Authorization": {"Bearer test"}, "Content-Type": {"application/json"},
			}, handler.Request{Permissions: []string{}})
			require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
			tt.mutate(t, h, res.Body.Data.KeyId, p.AuthorizedWorkspaceID)
			request := httptest.NewRequest(http.MethodPost, "/", nil)
			request.Header.Set("Authorization", "Bearer "+res.Body.Data.Key)
			session := &zen.Session{}
			require.NoError(t, session.Init(httptest.NewRecorder(), request, 0))
			for range 2 {
				_, err := h.Keys.GetRootKey(t.Context(), session)
				require.Error(t, err)
				code, ok := fault.GetCode(err)
				require.True(t, ok)
				require.Equal(t, tt.code, code)
			}
		})
	}
}
