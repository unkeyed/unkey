package v2_ratelimit_multi_limit_test

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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_ratelimit_multi_limit"
)

// ratelimitAuditLogRow contains the persisted fields that the test guarantees.
type ratelimitAuditLogRow struct {
	Event       string   `ch:"event"`
	Description string   `ch:"description"`
	ActorType   string   `ch:"actor_type"`
	UserAgent   string   `ch:"user_agent"`
	TargetTypes []string `ch:"target_types"`
	TargetIDs   []string `ch:"target_ids"`
	TargetNames []string `ch:"target_names"`
}

// TestMultiLimit_WritesRootKeyAuditLogPerCheck guarantees that every check
// produces its own direct audit event, even with metrics disabled, and that
// only the check with an applied override names it.
func TestMultiLimit_WritesRootKeyAuditLogPerCheck(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})
	route := &handler.Handler{
		DB:              h.DB,
		RatelimitEvents: h.RatelimitEvents,
		AuditLogBuffer:  h.AuditLogBuffer,
		Ratelimit:       h.Ratelimit,
		NamespaceCache:  h.Caches.RatelimitNamespace,
		Auditlogs:       h.Auditlogs,
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	overriddenNamespaceID, overriddenNamespaceName := createNamespace(t, h)
	plainNamespaceID, plainNamespaceName := createNamespace(t, h)
	rootKey := h.CreateRootKey(workspace.ID,
		fmt.Sprintf("ratelimit.%s.limit", overriddenNamespaceID),
		fmt.Sprintf("ratelimit.%s.limit", plainNamespaceID),
	)
	identifier := uid.New("sensitive")
	overrideID := uid.New(uid.RatelimitOverridePrefix)
	ctx := context.Background()
	err := db.Query.InsertRatelimitOverride(ctx, h.DB.RW(), db.InsertRatelimitOverrideParams{
		ID:          overrideID,
		WorkspaceID: workspace.ID,
		NamespaceID: overriddenNamespaceID,
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
		{Namespace: overriddenNamespaceName, Identifier: identifier, Limit: 10, Duration: time.Minute.Milliseconds()},
		{Namespace: plainNamespaceName, Identifier: identifier, Limit: 10, Duration: time.Minute.Milliseconds()},
	})
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Equal(t, overrideID, res.Body.Data.Limits[0].OverrideId)
	require.Empty(t, res.Body.Data.Limits[1].OverrideId)

	require.Empty(t, h.FindAuditLogsByTargetID(ctx, t, overriddenNamespaceID))
	require.Empty(t, h.FindAuditLogsByTargetID(ctx, t, plainNamespaceID))

	rows := make([]ratelimitAuditLogRow, 0)
	query := "SELECT event, description, actor_type, user_agent, " +
		"`targets.type` AS target_types, `targets.id` AS target_ids, `targets.name` AS target_names " +
		"FROM default.audit_logs_raw_v1 WHERE workspace_id = ? AND event = ? " +
		"AND hasAny(`targets.id`, [?, ?]) ORDER BY length(`targets.id`) DESC"
	require.Eventually(t, func() bool {
		rows = rows[:0]
		err := h.ClickHouse.Conn().Select(ctx, &rows, query,
			workspace.ID,
			string(auditlog.RatelimitLimitEvent),
			overriddenNamespaceID,
			plainNamespaceID,
		)
		return err == nil && len(rows) == 2
	}, 15*time.Second, 100*time.Millisecond)

	for _, row := range rows {
		require.Equal(t, string(auditlog.RatelimitLimitEvent), row.Event)
		require.Equal(t, string(auditlog.RootKeyActor), row.ActorType)
		require.Equal(t, "audit-log-test", row.UserAgent)
	}

	require.Equal(t, "Applied rate limit to namespace "+overriddenNamespaceID, rows[0].Description)
	require.Equal(t, []string{
		string(auditlog.RatelimitNamespaceTargetType),
		string(auditlog.RatelimitOverrideTargetType),
	}, rows[0].TargetTypes)
	require.Equal(t, []string{overriddenNamespaceID, overrideID}, rows[0].TargetIDs)
	require.Equal(t, []string{overriddenNamespaceName, identifier}, rows[0].TargetNames)

	require.Equal(t, "Applied rate limit to namespace "+plainNamespaceID, rows[1].Description)
	require.Equal(t, []string{string(auditlog.RatelimitNamespaceTargetType)}, rows[1].TargetTypes)
	require.Equal(t, []string{plainNamespaceID}, rows[1].TargetIDs)
	require.Equal(t, []string{plainNamespaceName}, rows[1].TargetNames)
}
