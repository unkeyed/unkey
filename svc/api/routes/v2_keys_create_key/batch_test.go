package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_create_key"
)

type invalidCreateOutboxAuditService struct {
	auditlogs.AuditLogService
}

func (invalidCreateOutboxAuditService) PrepareOutboxRows(
	_ context.Context,
	_ []auditlog.AuditLog,
) ([]db.InsertClickhouseOutboxParams, error) {
	return []db.InsertClickhouseOutboxParams{{
		Version:     auditlog.OutboxVersionV1,
		WorkspaceID: "ws_invalid",
		EventID:     "evt_invalid",
		Payload:     []byte("not-json"),
		CreatedAt:   time.Now().UnixMilli(),
	}}, nil
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

	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: h.Resources().UserWorkspace.ID})
	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.create_key")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}
	request := handler.Request{ApiId: api.ID}

	response := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request)
	require.Equal(t, http.StatusInternalServerError, response.Status)

	var keyCount int
	err := h.DB.RO().QueryRowContext(t.Context(), "SELECT COUNT(*) FROM `keys` WHERE key_auth_id = ?", api.KeyAuthID).Scan(&keyCount)
	require.NoError(t, err)
	require.Zero(t, keyCount)

	// The failed batch discards its physical connection. The pool and route must
	// remain usable on a fresh connection.
	route.Auditlogs = h.Auditlogs
	response = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request)
	require.Equal(t, http.StatusOK, response.Status)
	err = h.DB.RO().QueryRowContext(t.Context(), "SELECT COUNT(*) FROM `keys` WHERE key_auth_id = ?", api.KeyAuthID).Scan(&keyCount)
	require.NoError(t, err)
	require.Equal(t, 1, keyCount)
}
