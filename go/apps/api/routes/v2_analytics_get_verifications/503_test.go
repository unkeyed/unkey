package handler

import (
	"context"
	"database/sql"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
)

func Test503_ClickHouseConnectionFailure(t *testing.T) {
	h := testutil.NewHarness(t)

	workspace := h.CreateWorkspace()
	_ = h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_analytics")

	// Set up ClickHouse workspace settings with invalid connection info
	now := h.Clock.Now().UnixMilli()
	err := db.Query.InsertClickhouseWorkspaceSettings(context.Background(), h.DB.RW(), db.InsertClickhouseWorkspaceSettingsParams{
		WorkspaceID:               workspace.ID,
		Username:                  workspace.ID,
		PasswordEncrypted:         "invalid_password", // Invalid password will cause connection failure
		QuotaDurationSeconds:      3600,
		MaxQueriesPerWindow:       1000,
		MaxExecutionTimePerWindow: 1800,
		MaxQueryExecutionTime:     30,
		MaxQueryMemoryBytes:       1_000_000_000,
		MaxQueryResultRows:        10_000_000,
		CreatedAt:                 now,
		UpdatedAt:                 sql.NullInt64{Valid: true, Int64: now},
	})
	require.NoError(t, err)

	route := &Handler{
		Logger:                     h.Logger,
		DB:                         h.DB,
		Keys:                       h.Keys,
		ClickHouse:                 h.ClickHouse,
		AnalyticsConnectionManager: h.AnalyticsConnectionManager,
		Caches:                     h.Caches,
	}
	h.Register(route)

	headers := http.Header{
		"Authorization": []string{"Bearer " + rootKey},
		"Content-Type":  []string{"application/json"},
	}

	req := Request{
		Query: "SELECT COUNT(*) FROM key_verifications_v1",
	}

	res := testutil.CallRoute[Request, Response](h, route, headers, req)
	require.Equal(t, 503, res.Status) // Invalid password causes connection failure (503)
}
