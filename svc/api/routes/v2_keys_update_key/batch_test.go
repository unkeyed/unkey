package handler_test

import (
	"context"
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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_update_key"
)

type invalidOutboxAuditService struct {
	auditlogs.AuditLogService
}

func newUpdateKeyHarness(t *testing.T) *testutil.Harness {
	t.Helper()
	return testutil.NewHarness(t, testutil.HarnessConfig{
		Redis:                 false,
		ClickHouse:            false,
		MultiStatementBatches: true,
	})
}

func (invalidOutboxAuditService) PrepareOutboxRows(
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

func TestUpdateKeyBatchRollsBackWhenAuditInsertFails(t *testing.T) {
	t.Parallel()

	h := newUpdateKeyHarness(t)
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
	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.update_key")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	request := handler.Request{
		KeyId: key.KeyID,
		Name:  nullable.NewNullableWithValue("after"),
	}
	response := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request)
	require.Equal(t, http.StatusInternalServerError, response.Status)

	stored, err := db.Query.FindKeyByID(t.Context(), h.DB.RO(), key.KeyID)
	require.NoError(t, err)
	require.Equal(t, "before", stored.Name.String)

	// The failed batch discards its physical connection. The pool and route must
	// remain usable on a fresh connection.
	route.Auditlogs = h.Auditlogs
	response = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, request)
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
