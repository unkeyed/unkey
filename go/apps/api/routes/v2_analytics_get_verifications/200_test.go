package handler

import (
	"context"
	"database/sql"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func Test200_Success(t *testing.T) {
	h := testutil.NewHarness(t)

	workspace := h.CreateWorkspace()
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_analytics")

	// Set up ClickHouse workspace settings with workspace.ID as username
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

	// Buffer some key verifications
	for i := 0; i < 5; i++ {
		h.ClickHouse.BufferKeyVerification(schema.KeyVerificationRequestV1{
			RequestID:   uid.New(uid.RequestPrefix),
			Time:        now - int64(i*1000),
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			KeyID:       uid.New(uid.KeyPrefix),
			Region:      "us-west-1",
			Outcome:     "VALID",
			IdentityID:  "",
			Tags:        []string{},
		})
	}

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
		Query: "SELECT COUNT(*) as count FROM key_verifications",
	}

	// Wait for buffered data to be available
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		res := testutil.CallRoute[Request, Response](h, route, headers, req)
		require.Equal(c, 200, res.Status)
		require.NotNil(c, res.Body)
		require.Len(c, res.Body.Data.Verifications, 1)
	}, 30*time.Second, time.Second)
}
