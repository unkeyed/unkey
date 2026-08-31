package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_verify_key"
)

// auditLogRow contains the fields needed to verify the persisted audit event.
type auditLogRow struct {
	Time        int64    `ch:"time"`
	InsertedAt  int64    `ch:"inserted_at"`
	Event       string   `ch:"event"`
	Description string   `ch:"description"`
	ActorType   string   `ch:"actor_type"`
	ActorID     string   `ch:"actor_id"`
	ActorName   string   `ch:"actor_name"`
	UserAgent   string   `ch:"user_agent"`
	TargetTypes []string `ch:"target_types"`
	TargetIDs   []string `ch:"target_ids"`
	TargetNames []string `ch:"target_names"`
}

// TestVerifyKey_WritesRootKeyAuditLog guarantees that verification audit logs
// flush from the API buffer directly to ClickHouse.
func TestVerifyKey_WritesRootKeyAuditLog(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})
	route := &handler.Handler{
		DB:               h.DB,
		Keys:             h.Keys,
		DirectAuditLogs:  h.DirectAuditLogs,
		KeyVerifications: h.KeyVerifications,
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "api.*.verify_key")
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})
	identity := h.CreateIdentity(seed.CreateIdentityRequest{
		WorkspaceID: workspace.ID,
		ExternalID:  "audit-log-identity",
	})
	key := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
		IdentityID:  &identity.ID,
	})

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		"Content-Type":  {"application/json"},
		"User-Agent":    {"audit-log-test"},
	}, handler.Request{Key: key.Key})
	require.Equal(t, http.StatusOK, res.Status)

	ctx := context.Background()
	rows := make([]auditLogRow, 0)
	query := "SELECT time, inserted_at, event, description, actor_type, actor_id, actor_name, user_agent, " +
		"`targets.type` AS target_types, `targets.id` AS target_ids, `targets.name` AS target_names " +
		"FROM default.audit_logs_raw_v1 WHERE workspace_id = ? AND event = ?"
	require.Eventually(t, func() bool {
		rows = rows[:0]
		err := h.ClickHouse.Conn().Select(ctx, &rows, query,
			workspace.ID,
			string(auditlog.KeyVerifyEvent),
		)
		return err == nil && len(rows) == 1
	}, 15*time.Second, 100*time.Millisecond)

	row := rows[0]
	require.Positive(t, row.InsertedAt)
	require.GreaterOrEqual(t, row.InsertedAt, row.Time)
	require.Equal(t, string(auditlog.KeyVerifyEvent), row.Event)
	require.Equal(t, fmt.Sprintf("Verified key %s", key.KeyID), row.Description)
	require.Equal(t, string(auditlog.RootKeyActor), row.ActorType)
	require.NotEmpty(t, row.ActorID)
	require.NotEmpty(t, row.ActorName)
	require.Equal(t, "audit-log-test", row.UserAgent)
	require.Equal(t, []string{
		string(auditlog.KeyResourceType),
		string(auditlog.IdentityResourceType),
	}, row.TargetTypes)
	require.Equal(t, []string{key.KeyID, identity.ID}, row.TargetIDs)
	require.Equal(t, []string{"test-key", identity.ExternalID}, row.TargetNames)
}
