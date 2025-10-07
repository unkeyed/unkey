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

func Test403_NoAnalyticsPermission(t *testing.T) {
	h := testutil.NewHarness(t)

	workspace := h.CreateWorkspace()
	_ = h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})

	// Create root key WITHOUT read_analytics permission
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_api")

	// Set up ClickHouse workspace settings
	now := h.Clock.Now().UnixMilli()
	err := db.Query.InsertClickhouseWorkspaceSettings(context.Background(), h.DB.RW(), db.InsertClickhouseWorkspaceSettingsParams{
		WorkspaceID:               workspace.ID,
		Username:                  workspace.ID,
		PasswordEncrypted:         "encrypted_password",
		QuotaDurationSeconds:      3600,
		MaxQueriesPerWindow:       1000,
		MaxExecutionTimePerWindow: 1800,
		MaxQueryExecutionTime:     30,
		MaxQueryMemoryBytes:       1_000_000_000,
		MaxQueryResultRows:        10_000_000,
		MaxRowsToRead:             10_000_000,
		CreatedAt:                 now,
		UpdatedAt:                 sql.NullInt64{Valid: true, Int64: now},
	})
	require.NoError(t, err)

	route := &Handler{
		Logger:     h.Logger,
		DB:         h.DB,
		Keys:       h.Keys,
		ClickHouse: h.ClickHouse,
		AnalyticsConnectionManager: h.AnalyticsConnectionManager,
		Caches:     h.Caches,
	}
	h.Register(route)

	headers := http.Header{
		"Authorization": []string{"Bearer " + rootKey},
		"Content-Type":  []string{"application/json"},
	}

	req := Request{
		Query: "SELECT COUNT(*) FROM key_verifications",
	}

	res := testutil.CallRoute[Request, Response](h, route, headers, req)
	require.Equal(t, 403, res.Status)
}

func Test403_WrongApiPermission(t *testing.T) {
	h := testutil.NewHarness(t)

	workspace := h.CreateWorkspace()
	api1 := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})
	api2 := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})

	// Create root key with permission only for api1
	rootKey := h.CreateRootKey(workspace.ID, "api."+api1.ID+".read_analytics")

	// Set up ClickHouse workspace settings
	now := h.Clock.Now().UnixMilli()
	err := db.Query.InsertClickhouseWorkspaceSettings(context.Background(), h.DB.RW(), db.InsertClickhouseWorkspaceSettingsParams{
		WorkspaceID:               workspace.ID,
		Username:                  workspace.ID,
		PasswordEncrypted:         "encrypted_password",
		QuotaDurationSeconds:      3600,
		MaxQueriesPerWindow:       1000,
		MaxExecutionTimePerWindow: 1800,
		MaxQueryExecutionTime:     30,
		MaxQueryMemoryBytes:       1_000_000_000,
		MaxQueryResultRows:        10_000_000,
		MaxRowsToRead:             10_000_000,
		CreatedAt:                 now,
		UpdatedAt:                 sql.NullInt64{Valid: true, Int64: now},
	})
	require.NoError(t, err)

	route := &Handler{
		Logger:     h.Logger,
		DB:         h.DB,
		Keys:       h.Keys,
		ClickHouse: h.ClickHouse,
		AnalyticsConnectionManager: h.AnalyticsConnectionManager,
		Caches:     h.Caches,
	}
	h.Register(route)

	headers := http.Header{
		"Authorization": []string{"Bearer " + rootKey},
		"Content-Type":  []string{"application/json"},
	}

	// Query filtering by api2 but user only has permission for api1
	req := Request{
		Query: "SELECT COUNT(*) FROM key_verifications WHERE apiId = '" + api2.ID + "'",
	}

	res := testutil.CallRoute[Request, Response](h, route, headers, req)
	require.Equal(t, 403, res.Status)
}
