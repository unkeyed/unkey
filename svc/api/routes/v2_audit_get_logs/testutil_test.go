package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

const auditBucket = "unkey_mutations"

// newRoute builds the handler wired to the harness dependencies.
func newRoute(h *testutil.Harness) *Handler {
	return &Handler{
		ClickHouse:  h.ClickHouse,
		DB:          h.DB,
		LimitsCache: h.Caches.WorkspaceLimits,
	}
}

func bearer(rootKey string) http.Header {
	return http.Header{
		"Authorization": []string{"Bearer " + rootKey},
		"Content-Type":  []string{"application/json"},
	}
}

// auditRow builds an AuditLogV1 for a workspace with deterministic content.
func auditRow(workspaceID, eventID string, timeMs, insertedAtMs int64) schema.AuditLogV1 {
	return schema.AuditLogV1{
		EventID:       eventID,
		Time:          timeMs,
		InsertedAt:    insertedAtMs,
		WorkspaceID:   workspaceID,
		Bucket:        auditBucket,
		Source:        "platform",
		Event:         "key.create",
		Description:   "created a key",
		ActorType:     "root_key",
		ActorID:       "root_" + eventID,
		ActorName:     "root key",
		ActorMeta:     json.RawMessage(`{"role":"admin"}`),
		RemoteIP:      "1.2.3.4",
		UserAgent:     "unkey-test/1.0",
		Meta:          json.RawMessage(`{"foo":"bar"}`),
		TargetTypes:   []string{"key"},
		TargetIDs:     []string{"key_" + eventID},
		TargetNames:   []string{"a key"},
		TargetMetas:   []json.RawMessage{json.RawMessage(`{"k":"v"}`)},
		CorrelationID: "corr_" + eventID,
	}
}

// seedAudit inserts rows and waits until they are visible for reads.
func seedAudit(t *testing.T, h *testutil.Harness, workspaceID string, rows []schema.AuditLogV1) {
	t.Helper()
	ctx := context.Background()
	require.NoError(t, h.ClickHouse.InsertAuditLogs(ctx, rows))
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		got, err := h.ClickHouse.ListAuditLogs(ctx, clickhouse.ListAuditLogsRequest{
			WorkspaceID: workspaceID,
			Bucket:      auditBucket,
			StartMs:     0,
			EndMs:       time.Now().Add(24 * time.Hour).UnixMilli(),
			Limit:       1000,
		})
		require.NoError(c, err)
		require.GreaterOrEqual(c, len(got), len(rows))
	}, 30*time.Second, time.Second)
}
