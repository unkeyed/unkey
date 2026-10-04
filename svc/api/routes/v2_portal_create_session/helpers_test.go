package handler_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/internal/portal"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

func insertKeyspacePortal(t *testing.T, h *testutil.Harness, workspaceID, slug, keyspaceID string) string {
	t.Helper()

	mapping := portal.Mapping{Type: portal.MappingTypeKeyspace, ID: keyspaceID}
	return h.SeedPortal(t, workspaceID, slug, slug, mapping, nil, nil).ID
}

func countPortalSessions(t *testing.T, h *testutil.Harness, workspaceID, externalID string) int {
	t.Helper()

	var count int
	require.NoError(t, h.DB.RO().QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM portal_sessions WHERE workspace_id = ? AND external_id = ?",
		workspaceID, externalID,
	).Scan(&count))
	return count
}

func countAuditEntriesMentioning(t *testing.T, h *testutil.Harness, workspaceID, needle string) int {
	t.Helper()

	rows, err := db.Query.ListClickhouseOutboxByWorkspace(context.Background(), h.DB.RO(), workspaceID)
	require.NoError(t, err)

	count := 0
	for _, row := range rows {
		if strings.Contains(string(row.Payload), needle) {
			count++
		}
	}
	return count
}
