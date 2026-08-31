package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_create_key"
)

type invalidCreateOutboxAuditService struct {
	auditlogs.AuditLogService
}

func (invalidCreateOutboxAuditService) PrepareOutboxRows(
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

func TestCreateKeyBatchRollsBackWhenAuditInsertFails(t *testing.T) {
	t.Parallel()

	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: invalidCreateOutboxAuditService{AuditLogService: h.Auditlogs},
		Vault:     h.Vault,
	}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspaceID, EncryptedKeys: true})
	existingPermission := h.CreatePermission(seed.CreatePermissionRequest{
		WorkspaceID: workspaceID,
		Name:        "existing.read",
		Slug:        "existing.read",
	})
	role := h.CreateRole(seed.CreateRoleRequest{WorkspaceID: workspaceID, Name: "Batch Role"})
	rootKey := h.CreateRootKey(workspaceID, "api.*.create_key", "api.*.encrypt_key")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}
	externalID := "rollback-user"
	newPermissionSlug := "new.write"
	request := handler.Request{
		ApiId:       api.ID,
		ExternalId:  &externalID,
		Recoverable: ptr.P(true),
		Permissions: ptr.P([]string{existingPermission.Slug, newPermissionSlug}),
		Roles:       ptr.P([]string{role.Name}),
		Ratelimits: ptr.P([]openapi.RatelimitRequest{{
			Name:      "requests",
			Limit:     10,
			Duration:  1_000,
			AutoApply: true,
		}}),
	}

	var outboxCountBefore int
	err := h.DB.RO().QueryRowContext(t.Context(), "SELECT COUNT(*) FROM clickhouse_outbox WHERE workspace_id = ?", workspaceID).Scan(&outboxCountBefore)
	require.NoError(t, err)

	response := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request)
	require.Equal(t, http.StatusInternalServerError, response.Status)

	var keyCount int
	err = h.DB.RO().QueryRowContext(t.Context(), "SELECT COUNT(*) FROM `keys` WHERE key_auth_id = ?", api.KeyAuthID).Scan(&keyCount)
	require.NoError(t, err)
	require.Zero(t, keyCount)

	var identityCount int
	err = h.DB.RO().QueryRowContext(t.Context(), "SELECT COUNT(*) FROM identities WHERE workspace_id = ? AND external_id = ?", workspaceID, externalID).Scan(&identityCount)
	require.NoError(t, err)
	require.Zero(t, identityCount)

	var permissionCount int
	err = h.DB.RO().QueryRowContext(t.Context(), "SELECT COUNT(*) FROM permissions WHERE workspace_id = ? AND slug = ?", workspaceID, newPermissionSlug).Scan(&permissionCount)
	require.NoError(t, err)
	require.Zero(t, permissionCount)

	var ratelimitCount int
	err = h.DB.RO().QueryRowContext(t.Context(), "SELECT COUNT(*) FROM ratelimits WHERE workspace_id = ? AND name = 'requests'", workspaceID).Scan(&ratelimitCount)
	require.NoError(t, err)
	require.Zero(t, ratelimitCount)

	var encryptedCount int
	err = h.DB.RO().QueryRowContext(t.Context(), "SELECT COUNT(*) FROM encrypted_keys WHERE workspace_id = ?", workspaceID).Scan(&encryptedCount)
	require.NoError(t, err)
	require.Zero(t, encryptedCount)

	var outboxCountAfter int
	err = h.DB.RO().QueryRowContext(t.Context(), "SELECT COUNT(*) FROM clickhouse_outbox WHERE workspace_id = ?", workspaceID).Scan(&outboxCountAfter)
	require.NoError(t, err)
	require.Equal(t, outboxCountBefore, outboxCountAfter)

	// The failed batch discards its physical connection. The pool and route must
	// remain usable on a fresh connection.
	route.Auditlogs = h.Auditlogs
	response = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request)
	require.Equal(t, http.StatusOK, response.Status)
	err = h.DB.RO().QueryRowContext(t.Context(), "SELECT COUNT(*) FROM `keys` WHERE key_auth_id = ?", api.KeyAuthID).Scan(&keyCount)
	require.NoError(t, err)
	require.Equal(t, 1, keyCount)

	key, err := db.Query.FindKeyByID(t.Context(), h.DB.RO(), response.Body.Data.KeyId)
	require.NoError(t, err)
	require.True(t, key.IdentityID.Valid)
	require.NotEqual(t, sql.NullString{}, key.IdentityID)
}
