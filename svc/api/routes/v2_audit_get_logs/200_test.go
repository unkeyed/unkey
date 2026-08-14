package handler

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

func Test200_HappyPath(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})
	workspace := h.CreateWorkspace()
	h.SetupAnalytics(workspace.ID) // seeds a 30-day audit retention limit
	rootKey := h.CreateRootKey(workspace.ID, "audit.*.read_audit_log")

	base := time.Now().Add(-time.Hour).UnixMilli()
	seedAudit(t, h, workspace.ID, []schema.AuditLogV1{
		auditRow(workspace.ID, "evt_a", base+10, base+100),
	})

	route := newRoute(h)
	h.Register(route)

	res := testutil.CallRoute[Request, Response](h, route, bearer(rootKey), Request{})
	require.Equal(t, http.StatusOK, res.Status)
	require.Len(t, res.Body.Data, 1)

	got := res.Body.Data[0]
	require.Equal(t, "evt_a", got.AuditLogId)
	require.Equal(t, "v1", got.Version)
	require.Equal(t, "key.create", got.Event)
	require.Equal(t, "root_key", got.Actor.Type)
	require.NotNil(t, got.Context.IpAddress)
	require.Equal(t, "1.2.3.4", *got.Context.IpAddress)
	require.Len(t, got.Resources, 1)
	// time serializes as RFC3339 UTC
	require.Equal(t, time.UnixMilli(base+10).UTC(), got.Time)
	require.False(t, res.Body.Pagination.HasMore)
}

func Test200_PaginationIsGapFree(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})
	workspace := h.CreateWorkspace()
	h.SetupAnalytics(workspace.ID)
	rootKey := h.CreateRootKey(workspace.ID, "audit.*.read_audit_log")

	base := time.Now().Add(-time.Hour).UnixMilli()
	const total = 5
	rows := make([]schema.AuditLogV1, total)
	for i := range rows {
		// distinct inserted_at values, ascending
		rows[i] = auditRow(workspace.ID, fmt.Sprintf("evt_%02d", i), base+int64(i*10), base+int64(100+i*10))
	}
	seedAudit(t, h, workspace.ID, rows)

	route := newRoute(h)
	h.Register(route)

	// Page through with limit=2, following the cursor each time.
	seen := []string{}
	var cursor *string
	for page := 0; page < 10; page++ {
		req := Request{Limit: ptr.P(2), Cursor: cursor}
		res := testutil.CallRoute[Request, Response](h, route, bearer(rootKey), req)
		require.Equal(t, http.StatusOK, res.Status)
		for _, e := range res.Body.Data {
			seen = append(seen, e.AuditLogId)
		}
		if !res.Body.Pagination.HasMore {
			require.Nil(t, res.Body.Pagination.Cursor)
			break
		}
		require.NotNil(t, res.Body.Pagination.Cursor)
		cursor = res.Body.Pagination.Cursor
	}

	want := []string{"evt_00", "evt_01", "evt_02", "evt_03", "evt_04"}
	require.Equal(t, want, seen, "every event exactly once, in order, no gaps or dupes")
}

// Test200_LateDrainNoGap is the HTTP-level regression for the gap-free
// guarantee: a row that becomes visible with an older event time but a newer
// inserted_at than the consumer's watermark must still arrive on the next poll.
func Test200_LateDrainNoGap(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})
	workspace := h.CreateWorkspace()
	h.SetupAnalytics(workspace.ID)
	rootKey := h.CreateRootKey(workspace.ID, "audit.*.read_audit_log")
	route := newRoute(h)
	h.Register(route)

	base := time.Now().Add(-time.Hour).UnixMilli()
	early := auditRow(workspace.ID, "evt_early", base+50, base+100)
	seedAudit(t, h, workspace.ID, []schema.AuditLogV1{early})

	// Consume the first page, capture the cursor watermark.
	res := testutil.CallRoute[Request, Response](h, route, bearer(rootKey), Request{})
	require.Equal(t, http.StatusOK, res.Status)
	require.Len(t, res.Body.Data, 1)
	require.NotNil(t, res.Body.Pagination.Cursor)
	cursor := res.Body.Pagination.Cursor

	// A late-committing event: older event time, newer inserted_at.
	late := auditRow(workspace.ID, "evt_late", base+10, base+300)
	seedAudit(t, h, workspace.ID, []schema.AuditLogV1{early, late})

	res2 := testutil.CallRoute[Request, Response](h, route, bearer(rootKey), Request{Cursor: cursor})
	require.Equal(t, http.StatusOK, res2.Status)
	require.Len(t, res2.Body.Data, 1)
	require.Equal(t, "evt_late", res2.Body.Data[0].AuditLogId, "late-drained row must not be skipped")
}

func Test200_Filters(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})
	workspace := h.CreateWorkspace()
	h.SetupAnalytics(workspace.ID)
	rootKey := h.CreateRootKey(workspace.ID, "audit.*.read_audit_log")
	route := newRoute(h)
	h.Register(route)

	base := time.Now().Add(-time.Hour).UnixMilli()
	r1 := auditRow(workspace.ID, "evt_1", base+10, base+100)
	r2 := auditRow(workspace.ID, "evt_2", base+20, base+200)
	r2.Event = "api.delete"
	r2.ActorID = "root_beta"
	r2.TargetTypes = []string{"api"}
	seedAudit(t, h, workspace.ID, []schema.AuditLogV1{r1, r2})

	res := testutil.CallRoute[Request, Response](h, route, bearer(rootKey), Request{
		Event: ptr.P([]string{"api.delete"}),
	})
	require.Equal(t, http.StatusOK, res.Status)
	require.Len(t, res.Body.Data, 1)
	require.Equal(t, "evt_2", res.Body.Data[0].AuditLogId)
}

func Test200_RetentionClamp(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})
	workspace := h.CreateWorkspace()
	h.SetupAnalytics(workspace.ID, testutil.WithRetentionDays(7))
	rootKey := h.CreateRootKey(workspace.ID, "audit.*.read_audit_log")
	route := newRoute(h)
	h.Register(route)

	now := time.Now().UnixMilli()
	old := auditRow(workspace.ID, "evt_old", now-10*24*3600*1000, now-10*24*3600*1000) // 10 days ago
	recent := auditRow(workspace.ID, "evt_recent", now-3600*1000, now-3600*1000)       // 1 hour ago
	seedAudit(t, h, workspace.ID, []schema.AuditLogV1{old, recent})

	// A start older than the 7-day cutoff must be floored; only the recent row returns.
	start := time.UnixMilli(now - 30*24*3600*1000).UTC()
	res := testutil.CallRoute[Request, Response](h, route, bearer(rootKey), Request{Start: &start})
	require.Equal(t, http.StatusOK, res.Status)
	require.Len(t, res.Body.Data, 1)
	require.Equal(t, "evt_recent", res.Body.Data[0].AuditLogId)
}
