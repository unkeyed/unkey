package handler_test

import (
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/internal/services/keys/metrics"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_verify_key"
)

// TestReasonMetric sends a request down every verifyKey exit, including the
// early NOT_FOUND returns before Verify, and checks that each one is counted
// exactly once in unkey_key_verifications_total with its reason.
func TestReasonMetric(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB:               h.DB,
		Keys:             h.Keys,
		AuditLogBuffer:   h.AuditLogBuffer,
		KeyVerifications: h.KeyVerifications,
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	otherApi := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	rootKey := h.CreateRootKey(workspace.ID, "api.*.verify_key")
	scopedRootKey := h.CreateRootKey(workspace.ID, fmt.Sprintf("api.%s.verify_key", otherApi.ID))
	unknownKey := uid.New("unknown")

	count := func(t *testing.T, code openapi.V2KeysVerifyKeyResponseDataCode, reason keys.Reason) float64 {
		t.Helper()
		var metric dto.Metric
		require.NoError(t, metrics.KeyVerificationsTotal.WithLabelValues("key", string(code), string(reason)).Write(&metric))
		return metric.GetCounter().GetValue()
	}

	for _, tt := range []struct {
		name string
		// request creates what the case needs and returns the root key and body to send.
		request func(t *testing.T) (string, handler.Request)
		code    openapi.V2KeysVerifyKeyResponseDataCode
		reason  keys.Reason
	}{
		{
			name: "unknown hash",
			request: func(t *testing.T) (string, handler.Request) {
				return rootKey, handler.Request{Key: unknownKey}
			},
			code:   openapi.NOTFOUND,
			reason: keys.ReasonUnknownHash,
		},
		{
			name: "cached unknown hash",
			request: func(t *testing.T) (string, handler.Request) {
				return rootKey, handler.Request{Key: unknownKey}
			},
			code:   openapi.NOTFOUND,
			reason: keys.ReasonUnknownHash,
		},
		{
			// The counter records the hash lookup in Get, which misses before the
			// migration lookup runs.
			name: "unknown migration",
			request: func(t *testing.T) (string, handler.Request) {
				migrationID := uid.New("migration")
				return rootKey, handler.Request{Key: uid.New("migrated"), MigrationId: &migrationID}
			},
			code:   openapi.NOTFOUND,
			reason: keys.ReasonUnknownHash,
		},
		{
			name: "key in another workspace",
			request: func(t *testing.T) (string, handler.Request) {
				otherWorkspace := h.CreateWorkspace()
				foreignApi := h.CreateApi(seed.CreateApiRequest{WorkspaceID: otherWorkspace.ID})
				key := h.CreateKey(seed.CreateKeyRequest{WorkspaceID: otherWorkspace.ID, KeySpaceID: foreignApi.KeyAuthID.String})
				return rootKey, handler.Request{Key: key.Key}
			},
			code:   openapi.NOTFOUND,
			reason: keys.ReasonWorkspaceMismatch,
		},
		{
			name: "deleted api",
			request: func(t *testing.T) (string, handler.Request) {
				deletedApi := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
				key := h.CreateKey(seed.CreateKeyRequest{WorkspaceID: workspace.ID, KeySpaceID: deletedApi.KeyAuthID.String})
				require.NoError(t, db.Query.SoftDeleteApi(t.Context(), h.DB.RW(), db.SoftDeleteApiParams{
					Now:   sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
					ApiID: deletedApi.ID,
				}))
				return rootKey, handler.Request{Key: key.Key}
			},
			code:   openapi.NOTFOUND,
			reason: keys.ReasonAPIDeleted,
		},
		{
			name: "root key may not verify this api",
			request: func(t *testing.T) (string, handler.Request) {
				key := h.CreateKey(seed.CreateKeyRequest{WorkspaceID: workspace.ID, KeySpaceID: api.KeyAuthID.String})
				return scopedRootKey, handler.Request{Key: key.Key}
			},
			code:   openapi.NOTFOUND,
			reason: keys.ReasonPermissionDenied,
		},
		{
			name: "keyspace not allowed",
			request: func(t *testing.T) (string, handler.Request) {
				key := h.CreateKey(seed.CreateKeyRequest{WorkspaceID: workspace.ID, KeySpaceID: api.KeyAuthID.String})
				keyspaces := []string{otherApi.KeyAuthID.String}
				return rootKey, handler.Request{Key: key.Key, Keyspaces: &keyspaces}
			},
			code:   openapi.NOTFOUND,
			reason: keys.ReasonKeyspaceMismatch,
		},
		{
			name: "disabled key",
			request: func(t *testing.T) (string, handler.Request) {
				key := h.CreateKey(seed.CreateKeyRequest{WorkspaceID: workspace.ID, KeySpaceID: api.KeyAuthID.String, Disabled: true})
				return rootKey, handler.Request{Key: key.Key}
			},
			code:   openapi.DISABLED,
			reason: keys.ReasonDisabled,
		},
		{
			name: "valid key",
			request: func(t *testing.T) (string, handler.Request) {
				key := h.CreateKey(seed.CreateKeyRequest{WorkspaceID: workspace.ID, KeySpaceID: api.KeyAuthID.String})
				return rootKey, handler.Request{Key: key.Key}
			},
			code:   openapi.VALID,
			reason: keys.ReasonNone,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			root, req := tt.request(t)
			before := count(t, tt.code, tt.reason)

			headers := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {"Bearer " + root},
			}
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, http.StatusOK, res.Status, "%s", res.RawBody)
			require.Equal(t, tt.code, res.Body.Data.Code)

			require.Equal(t, before+1, count(t, tt.code, tt.reason))
		})
	}
}
