package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_update_key"
)

type invalidOutboxAuditService struct {
	auditlogs.AuditLogService
}

func (invalidOutboxAuditService) PrepareOutboxRows(
	ctx context.Context,
	logs []auditlog.AuditLog,
) ([]db.InsertClickhouseOutboxParams, error) {
	rows, err := auditlogs.PrepareOutboxRows(ctx, logs)
	if err != nil {
		return nil, err
	}
	rows[len(rows)-1].Payload = []byte("not-json")
	return rows, nil
}

func TestUpdateKeyBatchRollsBackWhenAuditInsertFails(t *testing.T) {
	t.Parallel()

	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB:           h.DB,
		Auditlogs:    invalidOutboxAuditService{AuditLogService: h.Auditlogs},
		KeyCache:     h.Caches.VerificationKeyByHash,
		UsageLimiter: h.UsageLimiter,
	}
	h.Register(route)

	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: h.Resources().UserWorkspace.ID})
	key := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: api.WorkspaceID,
		KeySpaceID:  api.KeyAuthID.String,
		Name:        ptr.P("before"),
	})
	role := h.CreateRole(seed.CreateRoleRequest{WorkspaceID: api.WorkspaceID, Name: "batch-role"})
	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.update_key")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	request := handler.Request{
		KeyId:      key.KeyID,
		Name:       nullable.NewNullableWithValue("after"),
		ExternalId: nullable.NewNullableWithValue("batch-identity"),
		Enabled:    ptr.P(false),
		Meta:       nullable.NewNullableWithValue(map[string]any{"batch": true}),
		Expires:    nullable.NewNullableWithValue(time.Now().Add(time.Hour).UnixMilli()),
		Credits: nullable.NewNullableWithValue(openapi.UpdateKeyCreditsData{
			Remaining: nullable.NewNullableWithValue(int64(99)),
		}),
		Ratelimits: nullable.NewNullableWithValue([]openapi.RatelimitRequest{{
			Name: "batch-limit", Limit: 10, Duration: 1_000, AutoApply: true,
		}}),
		Permissions: ptr.P([]string{"batch.permission"}),
		Roles:       ptr.P([]string{role.Name}),
	}
	response := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request)
	require.Equal(t, http.StatusInternalServerError, response.Status)

	stored, err := db.Query.FindKeyByID(t.Context(), h.DB.RO(), key.KeyID)
	require.NoError(t, err)
	require.Equal(t, "before", stored.Name.String)
	require.True(t, stored.Enabled)
	require.False(t, stored.IdentityID.Valid)
	require.False(t, stored.Meta.Valid)
	require.False(t, stored.Expires.Valid)
	require.False(t, stored.RemainingRequests.Valid)

	identities, err := db.Query.FindIdentity(t.Context(), h.DB.RO(), db.FindIdentityParams{
		WorkspaceID: api.WorkspaceID,
		Identity:    "batch-identity",
		Deleted:     false,
	})
	require.Error(t, err)
	require.Empty(t, identities.ID)

	ratelimits, err := db.Query.ListRatelimitsByKeyID(t.Context(), h.DB.RO(), sql.NullString{String: key.KeyID, Valid: true})
	require.NoError(t, err)
	require.Empty(t, ratelimits)
	permissions, err := db.Query.ListDirectPermissionsByKeyID(t.Context(), h.DB.RO(), key.KeyID)
	require.NoError(t, err)
	require.Empty(t, permissions)
	roles, err := db.Query.ListRolesByKeyID(t.Context(), h.DB.RO(), key.KeyID)
	require.NoError(t, err)
	require.Empty(t, roles)

	// The failed batch discards its physical connection. The pool and route must
	// remain usable on a fresh connection.
	route.Auditlogs = h.Auditlogs
	response = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		KeyId: key.KeyID,
		Name:  nullable.NewNullableWithValue("after"),
	})
	require.Equal(t, http.StatusOK, response.Status)
	stored, err = db.Query.FindKeyByID(t.Context(), h.DB.RO(), key.KeyID)
	require.NoError(t, err)
	require.Equal(t, "after", stored.Name.String)

	var auditRows int
	err = h.DB.RO().QueryRowContext(t.Context(), `
		SELECT COUNT(*)
		FROM clickhouse_outbox
		WHERE JSON_UNQUOTE(JSON_EXTRACT(payload, '$.event')) = ?
		  AND JSON_CONTAINS(payload, JSON_OBJECT('id', ?), '$.targets')
	`, string(auditlog.KeyUpdateEvent), key.KeyID).Scan(&auditRows)
	require.NoError(t, err)
	require.Equal(t, 1, auditRows)
}
