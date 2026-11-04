package handler_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/internal/services/keys/metrics"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_verify_key"
)

// exitCall is the request one exit path sends, and the key the hash matches.
type exitCall struct {
	root           string
	req            handler.Request
	keyID          string
	keyWorkspaceID string
}

// TestExitPaths sends a request down every verifyKey exit, including the early
// NOT_FOUND returns before Verify. Each exit must be counted exactly once in
// unkey_key_verifications_total with its reason, and each exit that is not
// VALID must buffer exactly one failure row that holds no key material.
func TestExitPaths(t *testing.T) {
	h := testutil.NewHarness(t)

	var mu sync.Mutex
	var rows []schema.KeyVerificationFailure
	buffer := batch.New(batch.Config[schema.KeyVerificationFailure]{
		Name:          "key_verification_failures_route_test",
		Drop:          false,
		BatchSize:     1,
		BufferSize:    1_000,
		FlushInterval: 10 * time.Millisecond,
		Consumers:     1,
		Flush: func(_ context.Context, batch []schema.KeyVerificationFailure) {
			mu.Lock()
			defer mu.Unlock()
			rows = append(rows, batch...)
		},
	})
	t.Cleanup(buffer.Close)

	route := &handler.Handler{
		DB:               h.DB,
		Keys:             h.Keys,
		AuditLogBuffer:   h.AuditLogBuffer,
		KeyVerifications: h.KeyVerifications,
		KeyVerificationFailures: keys.NewFailureRecorder(keys.FailureRecorderConfig{
			Buffer:            buffer,
			FingerprintSecret: "route-test-key-fingerprint-secret-0123456789",
			RowsPerSecond:     1_000,
			Clock:             clock.New(),
		}),
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	otherApi := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	rootKey := h.CreateRootKey(workspace.ID, "api.*.verify_key")
	scopedRootKey := h.CreateRootKey(workspace.ID, fmt.Sprintf("api.%s.verify_key", otherApi.ID))
	unknownKey := uid.New("unknown")

	rootKeyID := func(t *testing.T, root string) string {
		t.Helper()
		id, err := db.Query.FindKeyIDByHash(t.Context(), h.DB.RO(), hash.Sha256(root))
		require.NoError(t, err)
		return id
	}

	count := func(t *testing.T, code openapi.V2KeysVerifyKeyResponseDataCode, reason keys.Reason) float64 {
		t.Helper()
		var metric dto.Metric
		require.NoError(t, metrics.KeyVerificationsTotal.WithLabelValues("key", string(code), string(reason)).Write(&metric))
		return metric.GetCounter().GetValue()
	}

	// takeRows waits for the rows buffered by one request, then clears them.
	takeRows := func(t *testing.T, want int) []schema.KeyVerificationFailure {
		t.Helper()
		require.Eventually(t, func() bool {
			mu.Lock()
			defer mu.Unlock()
			return len(rows) >= want
		}, 5*time.Second, 10*time.Millisecond)
		// Give a wrongly duplicated row time to arrive before checking the count.
		time.Sleep(50 * time.Millisecond)
		mu.Lock()
		defer mu.Unlock()
		got := rows
		rows = nil
		require.Len(t, got, want)
		return got
	}

	requireNoKeyMaterial := func(t *testing.T, row schema.KeyVerificationFailure, rawKey string) {
		t.Helper()
		sum := sha256.Sum256([]byte(rawKey))
		encoded, err := json.Marshal(row)
		require.NoError(t, err)
		for _, secret := range []string{rawKey, base64.StdEncoding.EncodeToString(sum[:]), hex.EncodeToString(sum[:])} {
			require.NotContains(t, string(encoded), secret)
		}
		require.Equal(t, uint16(len(rawKey)), row.KeyLength)
		require.NotZero(t, row.KeyFingerprint)
	}

	for _, tt := range []struct {
		name string
		// request creates what the case needs and returns the call to send.
		request func(t *testing.T) exitCall
		code    openapi.V2KeysVerifyKeyResponseDataCode
		reason  keys.Reason
		// rowReason is the failure row's reason, or empty when no row is expected.
		rowReason   keys.Reason
		cacheResult string
	}{
		{
			name: "unknown hash",
			request: func(t *testing.T) exitCall {
				return exitCall{root: rootKey, req: handler.Request{Key: unknownKey}}
			},
			code:        openapi.NOTFOUND,
			reason:      keys.ReasonUnknownHash,
			rowReason:   keys.ReasonUnknownHash,
			cacheResult: "miss",
		},
		{
			name: "cached unknown hash",
			request: func(t *testing.T) exitCall {
				return exitCall{root: rootKey, req: handler.Request{Key: unknownKey}}
			},
			code:        openapi.NOTFOUND,
			reason:      keys.ReasonUnknownHash,
			rowReason:   keys.ReasonUnknownHash,
			cacheResult: "null",
		},
		{
			// The counter records the hash lookup in Get, which misses before the
			// migration lookup runs. The row records the migration lookup.
			name: "unknown migration",
			request: func(t *testing.T) exitCall {
				migrationID := uid.New("migration")
				return exitCall{root: rootKey, req: handler.Request{Key: uid.New("migrated"), MigrationId: &migrationID}}
			},
			code:      openapi.NOTFOUND,
			reason:    keys.ReasonUnknownHash,
			rowReason: keys.ReasonMigrationNotFound,
		},
		{
			name: "key in another workspace",
			request: func(t *testing.T) exitCall {
				otherWorkspace := h.CreateWorkspace()
				foreignApi := h.CreateApi(seed.CreateApiRequest{WorkspaceID: otherWorkspace.ID})
				key := h.CreateKey(seed.CreateKeyRequest{WorkspaceID: otherWorkspace.ID, KeySpaceID: foreignApi.KeyAuthID.String})
				return exitCall{root: rootKey, req: handler.Request{Key: key.Key}, keyID: key.KeyID, keyWorkspaceID: otherWorkspace.ID}
			},
			code:      openapi.NOTFOUND,
			reason:    keys.ReasonWorkspaceMismatch,
			rowReason: keys.ReasonWorkspaceMismatch,
		},
		{
			name: "deleted api",
			request: func(t *testing.T) exitCall {
				deletedApi := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
				key := h.CreateKey(seed.CreateKeyRequest{WorkspaceID: workspace.ID, KeySpaceID: deletedApi.KeyAuthID.String})
				require.NoError(t, db.Query.SoftDeleteApi(t.Context(), h.DB.RW(), db.SoftDeleteApiParams{
					Now:   sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
					ApiID: deletedApi.ID,
				}))
				return exitCall{root: rootKey, req: handler.Request{Key: key.Key}, keyID: key.KeyID, keyWorkspaceID: workspace.ID}
			},
			code:      openapi.NOTFOUND,
			reason:    keys.ReasonAPIDeleted,
			rowReason: keys.ReasonAPIDeleted,
		},
		{
			name: "root key may not verify this api",
			request: func(t *testing.T) exitCall {
				key := h.CreateKey(seed.CreateKeyRequest{WorkspaceID: workspace.ID, KeySpaceID: api.KeyAuthID.String})
				return exitCall{root: scopedRootKey, req: handler.Request{Key: key.Key}, keyID: key.KeyID, keyWorkspaceID: workspace.ID}
			},
			code:      openapi.NOTFOUND,
			reason:    keys.ReasonPermissionDenied,
			rowReason: keys.ReasonPermissionDenied,
		},
		{
			name: "keyspace not allowed",
			request: func(t *testing.T) exitCall {
				key := h.CreateKey(seed.CreateKeyRequest{WorkspaceID: workspace.ID, KeySpaceID: api.KeyAuthID.String})
				keyspaces := []string{otherApi.KeyAuthID.String}
				return exitCall{root: rootKey, req: handler.Request{Key: key.Key, Keyspaces: &keyspaces}, keyID: key.KeyID, keyWorkspaceID: workspace.ID}
			},
			code:      openapi.NOTFOUND,
			reason:    keys.ReasonKeyspaceMismatch,
			rowReason: keys.ReasonKeyspaceMismatch,
		},
		{
			name: "disabled key",
			request: func(t *testing.T) exitCall {
				key := h.CreateKey(seed.CreateKeyRequest{WorkspaceID: workspace.ID, KeySpaceID: api.KeyAuthID.String, Disabled: true})
				return exitCall{root: rootKey, req: handler.Request{Key: key.Key}, keyID: key.KeyID, keyWorkspaceID: workspace.ID}
			},
			code:      openapi.DISABLED,
			reason:    keys.ReasonDisabled,
			rowReason: keys.ReasonDisabled,
		},
		{
			name: "valid key",
			request: func(t *testing.T) exitCall {
				key := h.CreateKey(seed.CreateKeyRequest{WorkspaceID: workspace.ID, KeySpaceID: api.KeyAuthID.String})
				return exitCall{root: rootKey, req: handler.Request{Key: key.Key}}
			},
			code:   openapi.VALID,
			reason: keys.ReasonNone,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			call := tt.request(t)
			before := count(t, tt.code, tt.reason)

			headers := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {"Bearer " + call.root},
				"User-Agent":    {"exit-paths-test"},
			}
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, call.req)
			require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
			require.Equal(t, tt.code, res.Body.Data.Code)

			require.Equal(t, before+1, count(t, tt.code, tt.reason))

			if tt.rowReason == "" {
				takeRows(t, 0)
				return
			}
			row := takeRows(t, 1)[0]
			require.Equal(t, string(tt.code), row.Outcome)
			require.Equal(t, string(tt.rowReason), row.Reason)
			require.Equal(t, call.keyID, row.KeyID)
			require.Equal(t, call.keyWorkspaceID, row.KeyWorkspaceID)
			if tt.cacheResult != "" {
				require.Equal(t, tt.cacheResult, row.CacheResult)
			}
			require.Equal(t, workspace.ID, row.WorkspaceID, "caller workspace, not the key owner")
			require.Equal(t, rootKeyID(t, call.root), row.RootKeyID)
			require.Equal(t, "exit-paths-test", row.UserAgent)
			require.Equal(t, "test", row.Region)
			require.Equal(t, schema.SourceAPI, row.Source)
			require.Equal(t, uint32(1), row.SampleWeight)
			require.NotEmpty(t, row.RequestID)
			requireNoKeyMaterial(t, row, call.req.Key)
		})
	}

	// The caller gets an error response, not a verification outcome, so the
	// request is neither counted nor recorded as a failure row.
	t.Run("unconfigured ratelimit is not counted as valid", func(t *testing.T) {
		key := h.CreateKey(seed.CreateKeyRequest{WorkspaceID: workspace.ID, KeySpaceID: api.KeyAuthID.String})
		before := count(t, openapi.VALID, keys.ReasonNone)

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer " + rootKey},
		}
		req := handler.Request{
			Key:        key.Key,
			Ratelimits: &[]openapi.KeysVerifyKeyRatelimit{{Name: "does-not-exist"}},
		}
		res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusPreconditionFailed, res.Status, "%s", res.RawBody)

		require.Equal(t, before, count(t, openapi.VALID, keys.ReasonNone))
		takeRows(t, 0)
	})
}
