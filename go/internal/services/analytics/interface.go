package analytics

import (
	"context"

	"github.com/unkeyed/unkey/go/pkg/clickhouse"
	"github.com/unkeyed/unkey/go/pkg/db"
)

// ConnectionManager is the interface for managing per-workspace ClickHouse connections for analytics
type ConnectionManager interface {
	GetConnection(ctx context.Context, workspaceID string) (clickhouse.ClickHouse, db.FindClickhouseWorkspaceSettingsByWorkspaceIDRow, error)
}
