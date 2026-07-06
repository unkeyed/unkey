package clickhouse

import (
	"context"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/unkeyed/unkey/pkg/fault"
)

// IdentityBillableUsage is one end-user identity's billable usage for a single
// month, read from the per-identity rollup tables. Quantities only — pricing is
// applied downstream by the rate-card resolver.
type IdentityBillableUsage struct {
	WorkspaceID string
	IdentityID  string
	ExternalID  string
	// Verifications is the count of VALID key verifications.
	Verifications int64
	// SpentCredits is the sum of credits consumed by this identity's verifications.
	SpentCredits int64
	// RatelimitsPassed is the count of passed, identity-attributed standalone
	// ratelimit checks (from ratelimits_raw_v3). Bare-identifier ratelimits
	// that matched no identity are never attributed here.
	RatelimitsPassed int64
}

// GetBillableUsagePerIdentity returns per-end-user billable quantities for a
// workspace in a specific month, one row per identity. Usage is SCOPED to the
// enabled resources: verifications and credits are summed only over keyspaces
// in enabledKeyspaces, and passed ratelimits only over namespaces in
// enabledNamespaces (billing_billable_resources). An empty enabled set for a
// dimension contributes nothing; if both are empty the result is empty.
// Identities with zero billable usage in the month are absent from the result.
//
// Example:
//
//	usage, err := ch.GetBillableUsagePerIdentity(ctx, "ws_123abc", 2026, 7, keyspaces, namespaces)
//	if err != nil {
//	    return fmt.Errorf("failed to get per-identity usage: %w", err)
//	}
func (c *Client) GetBillableUsagePerIdentity(ctx context.Context, workspaceID string, year, month int, enabledKeyspaces, enabledNamespaces []string) ([]IdentityBillableUsage, error) {
	// Normalize nil to an empty slice so the Array(String) bind is well-formed;
	// `x IN []` then matches nothing, which is the correct "not enabled" result.
	if enabledKeyspaces == nil {
		enabledKeyspaces = []string{}
	}
	if enabledNamespaces == nil {
		enabledNamespaces = []string{}
	}

	// UNION ALL over the per-identity rollups, collapsed to one row per identity.
	// Each branch contributes its own metric and zeroes for the others, so the
	// outer GROUP BY sums into a single combined row. Verifications/credits are
	// filtered to enabled keyspaces; ratelimits to enabled namespaces.
	query := `
	SELECT
		identity_id,
		external_id,
		sum(verifications) AS verifications,
		sum(spent_credits) AS spent_credits,
		sum(ratelimits_passed) AS ratelimits_passed
	FROM (
		-- Literal zeroes are cast to Int64: bare 0 is UInt8, which would promote
		-- the summed column to UInt64 and break the signed scan on the Go side.
		SELECT identity_id, external_id, sum(count) AS verifications, toInt64(0) AS spent_credits, toInt64(0) AS ratelimits_passed
		FROM default.billable_verifications_per_identity_per_month_v2
		WHERE workspace_id = {workspace_id:String} AND year = {year:Int32} AND month = {month:Int32}
			AND key_space_id IN ({keyspaces:Array(String)})
		GROUP BY identity_id, external_id
		UNION ALL
		SELECT identity_id, external_id, toInt64(0), sum(spent_credits), toInt64(0)
		FROM default.billable_credits_per_identity_per_month_v2
		WHERE workspace_id = {workspace_id:String} AND year = {year:Int32} AND month = {month:Int32}
			AND key_space_id IN ({keyspaces:Array(String)})
		GROUP BY identity_id, external_id
		UNION ALL
		SELECT identity_id, external_id, toInt64(0), toInt64(0), sum(count)
		FROM default.billable_ratelimits_per_identity_per_month_v2
		WHERE workspace_id = {workspace_id:String} AND year = {year:Int32} AND month = {month:Int32}
			AND namespace_id IN ({namespaces:Array(String)})
		GROUP BY identity_id, external_id
	)
	GROUP BY identity_id, external_id
	HAVING verifications > 0 OR spent_credits > 0 OR ratelimits_passed > 0
	ORDER BY external_id, identity_id
	`

	rows, err := c.conn.Query(ctx, query,
		ch.Named("workspace_id", workspaceID),
		ch.Named("year", year),
		ch.Named("month", month),
		ch.Named("keyspaces", enabledKeyspaces),
		ch.Named("namespaces", enabledNamespaces),
	)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to query per-identity billable usage"))
	}
	defer func() { _ = rows.Close() }()

	var usage []IdentityBillableUsage
	for rows.Next() {
		row := IdentityBillableUsage{
			WorkspaceID:      workspaceID,
			IdentityID:       "",
			ExternalID:       "",
			Verifications:    0,
			SpentCredits:     0,
			RatelimitsPassed: 0,
		}
		if err := rows.Scan(&row.IdentityID, &row.ExternalID, &row.Verifications, &row.SpentCredits, &row.RatelimitsPassed); err != nil {
			return nil, fault.Wrap(err, fault.Internal("failed to scan per-identity usage row"))
		}
		usage = append(usage, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fault.Wrap(err, fault.Internal("error iterating per-identity usage rows"))
	}

	return usage, nil
}
