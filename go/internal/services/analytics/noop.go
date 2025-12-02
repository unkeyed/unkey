package analytics

import (
	"context"

	"github.com/unkeyed/unkey/go/pkg/clickhouse"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

// noopConnectionManager is a no-op implementation that returns errors indicating analytics is not configured
type noopConnectionManager struct{}

// NewNoopConnectionManager creates a new no-op connection manager for when analytics is not configured
func NewNoopConnectionManager() ConnectionManager {
	return &noopConnectionManager{}
}

// GetConnection always returns an error indicating analytics is not configured
func (m *noopConnectionManager) GetConnection(ctx context.Context, workspaceID string) (clickhouse.ClickHouse, db.FindClickhouseWorkspaceSettingsByWorkspaceIDRow, error) {
	return nil, db.FindClickhouseWorkspaceSettingsByWorkspaceIDRow{}, fault.New(
		"analytics not configured",
		fault.Code(codes.Data.Analytics.NotConfigured.URN()),
		fault.Public("Analytics are not configured for this instance"),
	)
}
