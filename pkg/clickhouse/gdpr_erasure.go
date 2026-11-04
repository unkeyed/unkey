package clickhouse

import (
	"context"
	"fmt"
	"slices"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/fault"
)

// erasureTable is a table a GDPR erasure removes rows from. mutation is set for
// tables with a projection: they reject a lightweight DELETE, so their rows are
// removed with an ALTER TABLE ... DELETE mutation instead.
type erasureTable struct {
	name     string
	mutation bool
}

var workspaceErasureTables = []erasureTable{
	{name: "audit_logs_raw_v1", mutation: true},
	{name: "frontline_requests_raw_v1", mutation: true},
	{name: "key_verifications_raw_v2", mutation: true},
	{name: "ratelimits_raw_v2", mutation: true},
	{name: "runtime_logs_raw_v1", mutation: true},
	{name: "active_workspaces_per_month_v2", mutation: false},
	{name: "api_requests_per_day_v2", mutation: false},
	{name: "api_requests_per_hour_v2", mutation: false},
	{name: "api_requests_per_minute_v2", mutation: false},
	{name: "api_requests_per_month_v2", mutation: false},
	{name: "api_requests_raw_v2", mutation: false},
	{name: "billable_ratelimits_per_month_v2", mutation: false},
	{name: "billable_verifications_per_month_v2", mutation: false},
	{name: "build_step_logs_v1", mutation: false},
	{name: "build_steps_v1", mutation: false},
	{name: "frontline_requests_per_15m_v1", mutation: false},
	{name: "frontline_requests_per_5m_v1", mutation: false},
	{name: "frontline_requests_per_day_v1", mutation: false},
	{name: "frontline_requests_per_hour_v1", mutation: false},
	{name: "frontline_requests_per_minute_v1", mutation: false},
	{name: "instance_checkpoints_v1", mutation: false},
	{name: "instance_events_raw_v1", mutation: false},
	{name: "instance_resources_per_15s_v1", mutation: false},
	{name: "instance_resources_per_day_v1", mutation: false},
	{name: "instance_resources_per_hour_v1", mutation: false},
	{name: "instance_resources_per_minute_v1", mutation: false},
	{name: "instance_resources_per_month_v1", mutation: false},
	{name: "instance_usage_per_hour_v1", mutation: false},
	{name: "key_last_used_v1", mutation: false},
	{name: "key_verifications_per_day_v3", mutation: false},
	{name: "key_verifications_per_hour_v3", mutation: false},
	{name: "key_verifications_per_minute_v3", mutation: false},
	{name: "key_verifications_per_month_v3", mutation: false},
	{name: "logdrain_deliveries_raw_v1", mutation: false},
	{name: "ratelimits_last_used_v2", mutation: false},
	{name: "ratelimits_per_day_v2", mutation: false},
	{name: "ratelimits_per_hour_v2", mutation: false},
	{name: "ratelimits_per_minute_v2", mutation: false},
	{name: "ratelimits_per_month_v2", mutation: false},
}

// endUserVerificationTables store an external_id next to the identity_id, so
// rows written before the identity existed are still found.
var endUserVerificationTables = []erasureTable{
	{name: "key_verifications_raw_v2", mutation: true},
	{name: "key_verifications_per_minute_v3", mutation: false},
	{name: "key_verifications_per_hour_v3", mutation: false},
	{name: "key_verifications_per_day_v3", mutation: false},
	{name: "key_verifications_per_month_v3", mutation: false},
}

var endUserLastUsedTable = erasureTable{name: "key_last_used_v1", mutation: false}

const (
	workspaceErasureFilter       = "workspace_id = ?"
	endUserVerificationFilter    = "workspace_id = ? AND (external_id = ? OR has(?, identity_id))"
	endUserLastUsedFilter        = "workspace_id = ? AND has(?, identity_id)"
	auditActorPseudonymizeFilter = "actor_id = ? AND (actor_name != '' OR remote_ip != '' OR user_agent != '' OR toJSONString(actor_meta) != '{}')"
)

// EndUserErasure identifies a customer's end user. IdentityIDs are the ids of
// every identity with the external id in the workspace.
type EndUserErasure struct {
	WorkspaceID string
	ExternalID  string
	IdentityIDs []string
}

// WorkspaceErasureTables returns the tables [Client.EraseWorkspaceRows] can
// erase, in the order a workspace erasure should run them.
func WorkspaceErasureTables() []string {
	return tableNames(workspaceErasureTables)
}

// EndUserErasureTables returns the tables [Client.EraseEndUserRows] can erase,
// in the order an end user erasure should run them.
func EndUserErasureTables() []string {
	return append(tableNames(endUserVerificationTables), endUserLastUsedTable.name)
}

// EraseWorkspaceRows deletes a workspace's rows from table, which must be one
// of [WorkspaceErasureTables], and returns how many rows matched.
func (c *Client) EraseWorkspaceRows(ctx context.Context, table, workspaceID string) (int64, error) {
	if err := assert.NotEmpty(workspaceID, "workspace id must not be empty"); err != nil {
		return 0, err
	}
	t, err := findErasureTable(workspaceErasureTables, table)
	if err != nil {
		return 0, err
	}
	return c.eraseRows(ctx, t, workspaceErasureFilter, workspaceID)
}

// EraseEndUserRows deletes an end user's rows from table, which must be one of
// [EndUserErasureTables], and returns how many rows matched.
func (c *Client) EraseEndUserRows(ctx context.Context, table string, endUser EndUserErasure) (int64, error) {
	if err := assert.All(
		assert.NotEmpty(endUser.WorkspaceID, "workspace id must not be empty"),
		assert.NotEmpty(endUser.ExternalID, "external id must not be empty"),
	); err != nil {
		return 0, err
	}
	if table == endUserLastUsedTable.name {
		return c.eraseRows(ctx, endUserLastUsedTable, endUserLastUsedFilter, endUser.WorkspaceID, endUser.IdentityIDs)
	}
	t, err := findErasureTable(endUserVerificationTables, table)
	if err != nil {
		return 0, err
	}
	return c.eraseRows(ctx, t, endUserVerificationFilter, endUser.WorkspaceID, endUser.ExternalID, endUser.IdentityIDs)
}

// PseudonymizeAuditActor blanks the name, metadata, address and user agent of
// every audit log row the actor wrote, keeping the actor id and the event, and
// returns how many rows it rewrote.
func (c *Client) PseudonymizeAuditActor(ctx context.Context, actorID string) (int64, error) {
	if err := assert.NotEmpty(actorID, "actor id must not be empty"); err != nil {
		return 0, err
	}
	return c.countThenRun(ctx, "audit_logs_raw_v1", auditActorPseudonymizeFilter,
		"ALTER TABLE audit_logs_raw_v1 UPDATE actor_name = '', actor_meta = '{}', remote_ip = '', user_agent = '' WHERE "+
			auditActorPseudonymizeFilter+" SETTINGS mutations_sync = 2",
		actorID,
	)
}

func (c *Client) eraseRows(ctx context.Context, table erasureTable, filter string, args ...any) (int64, error) {
	statement := "DELETE FROM " + table.name + " WHERE " + filter
	if table.mutation {
		statement = "ALTER TABLE " + table.name + " DELETE WHERE " + filter + " SETTINGS mutations_sync = 2"
	}
	return c.countThenRun(ctx, table.name, filter, statement, args...)
}

// countThenRun runs statement only when filter matches rows in table, so a
// repeated erasure does not queue mutations that rewrite nothing.
func (c *Client) countThenRun(ctx context.Context, table, filter, statement string, args ...any) (int64, error) {
	var matched uint64
	if err := c.conn.QueryRow(ctx, "SELECT count() FROM "+table+" WHERE "+filter, args...).Scan(&matched); err != nil {
		return 0, fault.Wrap(err, fault.Internal("failed to count rows to erase in "+table))
	}
	if matched == 0 {
		return 0, nil
	}
	if err := c.conn.Exec(ctx, statement, args...); err != nil {
		return 0, fault.Wrap(err, fault.Internal("failed to erase rows in "+table))
	}
	return int64(matched), nil
}

func findErasureTable(tables []erasureTable, name string) (erasureTable, error) {
	i := slices.IndexFunc(tables, func(t erasureTable) bool { return t.name == name })
	if i < 0 {
		return erasureTable{}, fmt.Errorf("%s is not an erasure table", name)
	}
	return tables[i], nil
}

func tableNames(tables []erasureTable) []string {
	names := make([]string, 0, len(tables))
	for _, t := range tables {
		names = append(names, t.name)
	}
	return names
}
