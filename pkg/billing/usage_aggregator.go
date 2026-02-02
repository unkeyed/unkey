package billing

import (
	"context"
	"time"

	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/fault"
)

// Usage is defined in end_user.go to avoid duplication

// UsageAggregator aggregates usage data from ClickHouse for billing purposes.
// It queries verification and rate limit events, filtering by external_id to
// associate usage with specific end users.
type UsageAggregator interface {
	// AggregateUsage aggregates usage from ClickHouse for a billing period.
	// Returns a map of external_id to Usage for all end users with usage in the period.
	// Events without external_id are excluded from results.
	AggregateUsage(ctx context.Context, workspaceID string, periodStart, periodEnd time.Time) (map[string]*Usage, error)

	// GetEndUserUsage gets usage for a specific end user in a billing period.
	// Returns zero usage if no events found for the external_id.
	GetEndUserUsage(ctx context.Context, workspaceID, externalID string, periodStart, periodEnd time.Time) (*Usage, error)
}

type usageAggregator struct {
	clickhouse clickhouse.Querier
}

// NewUsageAggregator creates a new UsageAggregator that queries ClickHouse.
func NewUsageAggregator(ch clickhouse.Querier) UsageAggregator {
	return &usageAggregator{
		clickhouse: ch,
	}
}

// AggregateUsage aggregates usage from ClickHouse for a billing period.
// It queries end_user_billable_verifications_per_month_v1 table,
// filtering by workspace_id, year, and month.
// Results are grouped by external_id.
//
// Requirements: 4.1, 4.2, 4.3, 4.4, 4.6
func (u *usageAggregator) AggregateUsage(ctx context.Context, workspaceID string, periodStart, periodEnd time.Time) (map[string]*Usage, error) {
	// Get year and month from the period
	year := int16(periodStart.Year())
	month := int8(periodStart.Month())

	// Query verifications from the end-user billable aggregated table
	verificationsQuery := `
		SELECT
			external_id,
			sum(count) as count
		FROM default.end_user_billable_verifications_per_month_v1
		WHERE workspace_id = ?
			AND year = ?
			AND month = ?
			AND external_id != ''
		GROUP BY external_id
	`

	verificationRows, err := u.clickhouse.QueryToMaps(ctx, verificationsQuery, workspaceID, year, month)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to query verifications for usage aggregation"))
	}

	// Query credits from the end-user billable aggregated table
	creditsQuery := `
		SELECT
			external_id,
			sum(count) as count
		FROM default.end_user_billable_credits_per_month_v1
		WHERE workspace_id = ?
			AND year = ?
			AND month = ?
			AND external_id != ''
		GROUP BY external_id
	`

	creditRows, err := u.clickhouse.QueryToMaps(ctx, creditsQuery, workspaceID, year, month)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to query credits for usage aggregation"))
	}

	// Build usage map
	usageMap := make(map[string]*Usage)

	// Process verifications
	for _, row := range verificationRows {
		externalID, ok := row["external_id"].(string)
		if !ok || externalID == "" {
			continue
		}

		count, ok := row["count"].(uint64)
		if !ok {
			continue
		}

		if _, exists := usageMap[externalID]; !exists {
			usageMap[externalID] = &Usage{
				ExternalID:     externalID,
				Verifications:  0,
				KeysWithAccess: 0,
				Credits:        0,
			}
		}
		usageMap[externalID].Verifications = int64(count)
	}

	// Process credits
	for _, row := range creditRows {
		externalID, ok := row["external_id"].(string)
		if !ok || externalID == "" {
			continue
		}

		count, ok := row["count"].(uint64)
		if !ok {
			continue
		}

		if _, exists := usageMap[externalID]; !exists {
			usageMap[externalID] = &Usage{
				ExternalID:     externalID,
				Verifications:  0,
				KeysWithAccess: 0,
				Credits:        0,
			}
		}
		usageMap[externalID].Credits = int64(count)
	}

	return usageMap, nil
}

// GetEndUserUsage gets usage for a specific end user in a billing period.
// Returns zero usage if no events found for the external_id.
//
// Requirements: 4.1, 4.2, 4.3, 4.4, 4.6
func (u *usageAggregator) GetEndUserUsage(ctx context.Context, workspaceID, externalID string, periodStart, periodEnd time.Time) (*Usage, error) {
	// Get year and month from the period
	year := int16(periodStart.Year())
	month := int8(periodStart.Month())

	usage := &Usage{
		ExternalID:     externalID,
		Verifications:  0,
		KeysWithAccess: 0,
		Credits:        0,
	}

	// Query verifications from the end-user billable aggregated table
	verificationsQuery := `
		SELECT sum(count) as count
		FROM default.end_user_billable_verifications_per_month_v1
		WHERE workspace_id = ?
			AND external_id = ?
			AND year = ?
			AND month = ?
	`

	var verificationCount uint64
	err := u.clickhouse.Conn().QueryRow(ctx, verificationsQuery, workspaceID, externalID, year, month).Scan(&verificationCount)
	if err != nil && err.Error() != "sql: no rows in result set" {
		return nil, fault.Wrap(err, fault.Internal("failed to query verifications for end user"))
	}
	usage.Verifications = int64(verificationCount)

	// Query credits from the end-user billable aggregated table
	creditsQuery := `
		SELECT sum(count) as count
		FROM default.end_user_billable_credits_per_month_v1
		WHERE workspace_id = ?
			AND external_id = ?
			AND year = ?
			AND month = ?
	`

	var creditsCount uint64
	err = u.clickhouse.Conn().QueryRow(ctx, creditsQuery, workspaceID, externalID, year, month).Scan(&creditsCount)
	if err != nil && err.Error() != "sql: no rows in result set" {
		return nil, fault.Wrap(err, fault.Internal("failed to query credits for end user"))
	}
	usage.Credits = int64(creditsCount)

	return usage, nil
}
