package v2RatelimitLimit_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_ratelimit_limit"
)

// ratelimitAuditLogRow contains the persisted fields that the test guarantees.
type ratelimitAuditLogRow struct {
	Time            int64    `ch:"time"`
	InsertedAt      int64    `ch:"inserted_at"`
	Event           string   `ch:"event"`
	Description     string   `ch:"description"`
	ActorType       string   `ch:"actor_type"`
	ActorID         string   `ch:"actor_id"`
	ActorName       string   `ch:"actor_name"`
	UserAgent       string   `ch:"user_agent"`
	MetaText        string   `ch:"meta_text"`
	TargetTypes     []string `ch:"target_types"`
	TargetIDs       []string `ch:"target_ids"`
	TargetNames     []string `ch:"target_names"`
	TargetsMetaText string   `ch:"targets_meta_text"`
}

// TestLimit_WritesRootKeyAuditLog guarantees that metrics opt-out does not stop
// direct audit logs. The event bypasses MySQL and names an applied override.
func TestLimit_WritesRootKeyAuditLog(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})
	route := &handler.Handler{
		DB:              h.DB,
		RatelimitEvents: h.RatelimitEvents,
		DirectAuditLogs: h.DirectAuditLogs,
		Ratelimit:       h.Ratelimit,
		NamespaceCache:  h.Caches.RatelimitNamespace,
		Auditlogs:       h.Auditlogs,
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	namespaceID, namespaceName := createNamespace(t, h)
	rootKey := h.CreateRootKey(workspace.ID, fmt.Sprintf("ratelimit.%s.limit", namespaceID))
	identifier := uid.New("sensitive")
	overrideID := uid.New(uid.RatelimitOverridePrefix)
	ctx := context.Background()
	err := db.Query.InsertRatelimitOverride(ctx, h.DB.RW(), db.InsertRatelimitOverrideParams{
		ID:          overrideID,
		WorkspaceID: workspace.ID,
		NamespaceID: namespaceID,
		Identifier:  identifier,
		Limit:       10,
		Duration:    uint64(time.Minute.Milliseconds()),
		CreatedAt:   h.Clock.Now().UnixMilli(),
	})
	require.NoError(t, err)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
		"Authorization":   {fmt.Sprintf("Bearer %s", rootKey)},
		"Content-Type":    {"application/json"},
		"User-Agent":      {"audit-log-test"},
		"X-Unkey-Metrics": {"disabled"},
	}, handler.Request{
		Namespace:  namespaceName,
		Identifier: identifier,
		Limit:      10,
		Duration:   time.Minute.Milliseconds(),
	})
	require.Equal(t, http.StatusOK, res.Status)
	require.Equal(t, overrideID, res.Body.Data.OverrideId)

	require.Empty(t, h.FindAuditLogsByTargetID(ctx, t, namespaceID))
	require.Empty(t, h.FindAuditLogsByTargetID(ctx, t, overrideID))

	rows := make([]ratelimitAuditLogRow, 0)
	query := "SELECT time, inserted_at, event, description, actor_type, actor_id, actor_name, user_agent, meta_text, " +
		"`targets.type` AS target_types, `targets.id` AS target_ids, `targets.name` AS target_names, targets_meta_text " +
		"FROM default.audit_logs_raw_v1 WHERE workspace_id = ? AND event = ? LIMIT 1"
	require.Eventually(t, func() bool {
		rows = rows[:0]
		err := h.ClickHouse.Conn().Select(ctx, &rows, query,
			workspace.ID,
			string(auditlog.RatelimitLimitEvent),
		)
		return err == nil && len(rows) == 1
	}, 15*time.Second, 100*time.Millisecond)

	row := rows[0]
	require.Positive(t, row.InsertedAt)
	require.GreaterOrEqual(t, row.InsertedAt, row.Time)
	require.Equal(t, string(auditlog.RatelimitLimitEvent), row.Event)
	require.Equal(t, "Applied rate limit to namespace "+namespaceID, row.Description)
	require.Equal(t, string(auditlog.RootKeyActor), row.ActorType)
	require.NotEmpty(t, row.ActorID)
	require.NotEmpty(t, row.ActorName)
	require.Equal(t, "audit-log-test", row.UserAgent)
	require.Equal(t, []string{
		string(auditlog.RatelimitNamespaceResourceType),
		string(auditlog.RatelimitOverrideResourceType),
	}, row.TargetTypes)
	require.Equal(t, []string{namespaceID, overrideID}, row.TargetIDs)
	require.Equal(t, []string{namespaceName, identifier}, row.TargetNames)
	require.JSONEq(t, `{}`, row.MetaText)
	require.Equal(t, "{} {}", row.TargetsMetaText)
	require.NotContains(t, row.MetaText, identifier)
	require.NotContains(t, row.TargetsMetaText, identifier)
}
