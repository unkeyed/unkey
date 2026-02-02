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
// It queries both key_verifications_raw_v2 and ratelimits_raw_v2 tables,
// filtering by workspace_id, time range, and excluding events without external_id.
// Results are grouped by external_id.
//
// Requirements: 4.1, 4.2, 4.3, 4.4, 4.6
func (u *usageAggregator) AggregateUsage(ctx context.Context, workspaceID string, periodStart, periodEnd time.Time) (map[string]*Usage, error) {
	// Convert times to milliseconds for ClickHouse
	startMs := periodStart.UnixMilli()
	endMs := periodEnd.UnixMilli()

	// Query verifications grouped by external_id
	verificationsQuery := `
		SELECT
			external_id,
			count() as count
		FROM default.key_verifications_raw_v2
		WHERE workspace_id = ?
			AND time >= ?
			AND time < ?
			AND external_id != ''
			AND outcome = 'VALID'
		GROUP BY external_id
	`

	verificationRows, err := u.clickhouse.QueryToMaps(ctx, verificationsQuery, workspaceID, startMs, endMs)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to query verifications for usage aggregation"))
	}

	// Query unique keys with access (VALID=true) grouped by external_id
	// This counts unique key_id values for billing key access charges
	keysWithAccessQuery := `
		SELECT
			external_id,
			count(distinct key_id) as count
		FROM default.key_verifications_raw_v2
		WHERE workspace_id = ?
			AND time >= ?
			AND time < ?
			AND external_id != ''
			AND outcome = 'VALID'
		GROUP BY external_id
	`

	keysWithAccessRows, err := u.clickhouse.QueryToMaps(ctx, keysWithAccessQuery, workspaceID, startMs, endMs)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to query keys with access for usage aggregation"))
	}

	// Query rate limits grouped by external_id
	// Note: Ratelimit table uses identifier field, not external_id
	// We need to join with identities to get external_id
	// For now, we'll query ratelimits by identifier and assume identifier matches external_id
	ratelimitsQuery := `
		SELECT
			identifier as external_id,
			count() as count
		FROM default.ratelimits_raw_v2
		WHERE workspace_id = ?
			AND time >= ?
			AND time < ?
			AND identifier != ''
			AND passed = true
		GROUP BY identifier
	`

	ratelimitRows, err := u.clickhouse.QueryToMaps(ctx, ratelimitsQuery, workspaceID, startMs, endMs)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to query ratelimits for usage aggregation"))
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
				RateLimits:     0,
				KeysWithAccess: 0,
			}
		}
		usageMap[externalID].Verifications = int64(count)
	}

	// Process keys with access
	for _, row := range keysWithAccessRows {
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
				RateLimits:     0,
				KeysWithAccess: 0,
			}
		}
		usageMap[externalID].KeysWithAccess = int64(count)
	}

	// Process rate limits
	for _, row := range ratelimitRows {
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
				RateLimits:     0,
				KeysWithAccess: 0,
			}
		}
		usageMap[externalID].RateLimits = int64(count)
	}

	return usageMap, nil
}

// GetEndUserUsage gets usage for a specific end user in a billing period.
// Returns zero usage if no events found for the external_id.
//
// Requirements: 4.1, 4.2, 4.3, 4.4, 4.6
func (u *usageAggregator) GetEndUserUsage(ctx context.Context, workspaceID, externalID string, periodStart, periodEnd time.Time) (*Usage, error) {
	// Convert times to milliseconds for ClickHouse
	startMs := periodStart.UnixMilli()
	endMs := periodEnd.UnixMilli()

	usage := &Usage{
		ExternalID:     externalID,
		Verifications:  0,
		RateLimits:     0,
		KeysWithAccess: 0,
	}

	// Query verifications for this external_id
	verificationsQuery := `
		SELECT count() as count
		FROM default.key_verifications_raw_v2
		WHERE workspace_id = ?
			AND external_id = ?
			AND time >= ?
			AND time < ?
			AND outcome = 'VALID'
	`

	var verificationCount uint64
	err := u.clickhouse.Conn().QueryRow(ctx, verificationsQuery, workspaceID, externalID, startMs, endMs).Scan(&verificationCount)
	if err != nil && err.Error() != "sql: no rows in result set" {
		return nil, fault.Wrap(err, fault.Internal("failed to query verifications for end user"))
	}
	usage.Verifications = int64(verificationCount)

	// Query unique keys with access for this external_id
	keysWithAccessQuery := `
		SELECT count(distinct key_id) as count
		FROM default.key_verifications_raw_v2
		WHERE workspace_id = ?
			AND external_id = ?
			AND time >= ?
			AND time < ?
			AND outcome = 'VALID'
	`

	var keysWithAccessCount uint64
	err = u.clickhouse.Conn().QueryRow(ctx, keysWithAccessQuery, workspaceID, externalID, startMs, endMs).Scan(&keysWithAccessCount)
	if err != nil && err.Error() != "sql: no rows in result set" {
		return nil, fault.Wrap(err, fault.Internal("failed to query keys with access for end user"))
	}
	usage.KeysWithAccess = int64(keysWithAccessCount)

	// Query rate limits for this external_id (using identifier field)
	ratelimitsQuery := `
		SELECT count() as count
		FROM default.ratelimits_raw_v2
		WHERE workspace_id = ?
			AND identifier = ?
			AND time >= ?
			AND time < ?
			AND passed = true
	`

	var ratelimitCount uint64
	err = u.clickhouse.Conn().QueryRow(ctx, ratelimitsQuery, workspaceID, externalID, startMs, endMs).Scan(&ratelimitCount)
	if err != nil && err.Error() != "sql: no rows in result set" {
		return nil, fault.Wrap(err, fault.Internal("failed to query ratelimits for end user"))
	}
	usage.RateLimits = int64(ratelimitCount)

	return usage, nil
}
