package analytics

import (
	"context"

	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/db"
)

// ConnectionManager is the interface for managing per-workspace ClickHouse connections for analytics
type ConnectionManager interface {
	GetConnection(ctx context.Context, workspaceID string) (clickhouse.ClickHouse, db.FindClickhouseWorkspaceSettingsByWorkspaceIDRow, error)
}
