package clickhouse

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	driver "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/unkeyed/unkey/pkg/logger"
)

const (
	// AnalyticsExecutionTimeMax is the hard server-side execution cap for customer analytics queries.
	AnalyticsExecutionTimeMax = 30
	// AnalyticsQueryTimeout bounds the ClickHouse phase within the API's larger
	// request timeout. clickhouse-go adds five seconds when converting this
	// deadline into max_execution_time, so a 10-second client timeout requests
	// 15 seconds from ClickHouse and remains below the 30-second server cap.
	AnalyticsQueryTimeout = 10 * time.Second
	// AnalyticsResultBytesMax is the maximum encoded size of customer analytics results.
	AnalyticsResultBytesMax = 4 << 20
	// AnalyticsASTDepthMax is the maximum ClickHouse AST depth for customer analytics queries.
	AnalyticsASTDepthMax = 100
	// AnalyticsASTElementsMax is the maximum number of ClickHouse AST elements for customer analytics queries.
	AnalyticsASTElementsMax = 2_000
)

var (
	// validIdentifier matches safe ClickHouse identifiers (usernames, policy names, quota names, profile names)
	// Allows alphanumeric characters and underscores only
	validIdentifier = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

	// validTableName matches safe ClickHouse table names in database.table format
	// Allows alphanumeric characters and underscores in both database and table parts
	validTableName = regexp.MustCompile(`^[a-zA-Z0-9_]+\.[a-zA-Z0-9_]+$`)

	// Table type patterns for retention filter generation
	rawTablePattern      = regexp.MustCompile(`_raw_v\d+$`)
	perMinuteHourPattern = regexp.MustCompile(`_per_minute_v\d+$|_per_hour_v\d+$`)
	perDayMonthPattern   = regexp.MustCompile(`_per_day_v\d+$|_per_month_v\d+$`)
)

// AllowedTable grants SELECT on one analytics table. An empty Columns slice
// grants all columns; a non-empty slice grants only the columns in the slice.
// Column grants are the only control on column access: the query parser checks
// tables and functions, but it does not check columns. Thus a column that is
// not in this list stays out of reach for all customer SQL.
type AllowedTable struct {
	Name    string
	Columns []string
}

// UserConfig contains configuration for creating/updating a ClickHouse user
type UserConfig struct {
	WorkspaceID string
	Username    string
	Password    string

	// Tables to grant SELECT permission on
	AllowedTables []AllowedTable

	// Quota settings (per window)
	QuotaDurationSeconds      int32
	MaxQueriesPerWindow       int32
	MaxExecutionTimePerWindow int32

	// Per-query limits (settings profile)
	MaxQueryExecutionTime int32
	MaxQueryMemoryBytes   int64
	MaxQueryResultRows    int32

	// Data retention in days, read from the limits table.
	RetentionDays int32
}

// validateIdentifiers checks that all identifiers in the config are safe to use in SQL statements.
// This prevents SQL injection since ClickHouse identifiers cannot be parameterized.
func validateIdentifiers(config UserConfig) error {
	// Validate username
	if !validIdentifier.MatchString(config.Username) {
		return fmt.Errorf("invalid username: must contain only alphanumeric characters and underscores, got %q", config.Username)
	}

	// Validate workspace ID (used in policy/quota/profile names and WHERE clauses)
	if !validIdentifier.MatchString(config.WorkspaceID) {
		return fmt.Errorf("invalid workspace_id: must contain only alphanumeric characters and underscores, got %q", config.WorkspaceID)
	}

	// Validate all table and column names
	for _, table := range config.AllowedTables {
		if !validTableName.MatchString(table.Name) {
			return fmt.Errorf("invalid table name: must be in format 'database.table' with alphanumeric characters and underscores only, got %q", table.Name)
		}

		for _, column := range table.Columns {
			if !validIdentifier.MatchString(column) {
				return fmt.Errorf("invalid column name on table %q: must contain only alphanumeric characters and underscores, got %q", table.Name, column)
			}
		}
	}

	return nil
}

// grantSelectSQL makes the SELECT grant for one table. ClickHouse cannot use
// query parameters for identifiers, thus this function puts each name directly
// into the statement. Each name must first pass validateIdentifiers.
func grantSelectSQL(table AllowedTable, username string) string {
	if len(table.Columns) == 0 {
		return fmt.Sprintf("GRANT SELECT ON %s TO %s", table.Name, username)
	}

	return fmt.Sprintf("GRANT SELECT(%s) ON %s TO %s", strings.Join(table.Columns, ", "), table.Name, username)
}

// getTimeRetentionFilter returns the appropriate retention filter based on table type and retention days.
// Different tables use different time column types:
// - Raw tables (_raw_v2): time Int64 (Unix milliseconds)
// - Per-minute/hour tables: time DateTime
// - Per-day/month tables: time Date
// All filters are rounded to the start of the day for consistency and predictability.
func getTimeRetentionFilter(tableName string, retentionDays int32) string {
	switch {
	case rawTablePattern.MatchString(tableName):
		// Raw tables use Int64 Unix milliseconds
		// Round to start of day for clean retention boundaries
		return fmt.Sprintf("time >= toUnixTimestamp(toStartOfDay(now() - INTERVAL %d DAY)) * 1000", retentionDays)
	case perMinuteHourPattern.MatchString(tableName):
		// Minute/hour aggregation tables use DateTime
		// Round to start of day for clean retention boundaries
		return fmt.Sprintf("time >= toStartOfDay(now() - INTERVAL %d DAY)", retentionDays)
	case perDayMonthPattern.MatchString(tableName):
		// Day/month aggregation tables use Date
		// today() - INTERVAL already gives start of day
		return fmt.Sprintf("time >= today() - INTERVAL %d DAY", retentionDays)
	default:
		// Default to DateTime format for unknown table types, rounded to start of day
		return fmt.Sprintf("time >= toStartOfDay(now() - INTERVAL %d DAY)", retentionDays)
	}
}

// ConfigureUser creates or updates a ClickHouse user with all necessary permissions, quotas, and settings.
// This is idempotent - it can be run multiple times to update settings.
func (c *Client) ConfigureUser(ctx context.Context, config UserConfig) error {
	logger.Info("configuring clickhouse user", "workspace_id", config.WorkspaceID, "username", config.Username)

	if config.MaxQueryResultRows <= 0 {
		return fmt.Errorf("query result row limit must be positive")
	}

	// Validate all identifiers to prevent SQL injection
	if err := validateIdentifiers(config); err != nil {
		return fmt.Errorf("identifier validation failed: %w", err)
	}

	// Create or alter ClickHouse user
	logger.Info("creating/updating clickhouse user")
	createUserSQL := fmt.Sprintf("CREATE USER IF NOT EXISTS %s IDENTIFIED WITH sha256_password BY {password:String}", config.Username)
	err := c.Exec(ctx, createUserSQL, driver.Named("password", config.Password))
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	// Revoke all permissions
	logger.Info("revoking all permissions")
	revokeSQL := fmt.Sprintf("REVOKE ALL ON *.* FROM %s", config.Username)
	err = c.Exec(ctx, revokeSQL)
	if err != nil {
		logger.Warn("failed to revoke permissions (user may be new)", "error", err)
	}

	// Grant SELECT on specified tables
	for _, table := range config.AllowedTables {
		logger.Debug("granting SELECT permission", "table", table.Name, "columns", len(table.Columns))
		err = c.Exec(ctx, grantSelectSQL(table, config.Username))
		if err != nil {
			return fmt.Errorf("failed to grant SELECT on %s: %w", table.Name, err)
		}
	}

	// Create row-level security (RLS) policies
	policyName := fmt.Sprintf("workspace_%s_rls", config.WorkspaceID)
	for _, table := range config.AllowedTables {
		logger.Debug("creating row policy", "table", table.Name, "policy", policyName, "retention_days", config.RetentionDays)

		// Build time retention filter based on table type and configured retention period
		timeFilter := getTimeRetentionFilter(table.Name, config.RetentionDays)

		createPolicySQL := fmt.Sprintf(
			"CREATE ROW POLICY OR REPLACE %s ON %s FOR SELECT USING workspace_id = '%s' AND (%s) TO %s",
			policyName, table.Name, config.WorkspaceID, timeFilter, config.Username,
		)
		err = c.Exec(ctx, createPolicySQL)
		if err != nil {
			return fmt.Errorf("failed to create row policy on %s: %w", table.Name, err)
		}
	}

	// Create or replace quota
	quotaName := fmt.Sprintf("workspace_%s_quota", config.WorkspaceID)
	logger.Info("creating/updating quota", "name", quotaName)

	createOrReplaceQuotaSQL := fmt.Sprintf(`
		CREATE QUOTA OR REPLACE %s
		FOR INTERVAL %d SECOND
			MAX queries = %d,
			MAX execution_time = %d
			-- MAX result_rows is intentionally NOT set here
			-- Per-window result row limits are too restrictive for analytics queries
			-- which legitimately need to return large result sets.
			-- Per-query limits are still enforced via the settings profile (max_result_rows).
		TO %s
	`,
		quotaName,
		config.QuotaDurationSeconds,
		config.MaxQueriesPerWindow,
		config.MaxExecutionTimePerWindow,
		config.Username,
	)
	err = c.Exec(ctx, createOrReplaceQuotaSQL)
	if err != nil {
		return fmt.Errorf("failed to create/replace quota: %w", err)
	}

	// Create or replace settings profile
	profileName := fmt.Sprintf("workspace_%s_profile", config.WorkspaceID)
	logger.Info("creating/updating settings profile", "name", profileName)

	createOrReplaceProfileSQL := fmt.Sprintf(`
		CREATE SETTINGS PROFILE OR REPLACE %s SETTINGS
			max_execution_time = %d MIN 1 MAX %d CHANGEABLE_IN_READONLY,
			max_memory_usage = %d READONLY,
			max_result_rows = %d READONLY,
			max_result_bytes = %d READONLY,
			result_overflow_mode = 'throw' READONLY,
			max_ast_depth = %d READONLY,
			max_ast_elements = %d READONLY,
			readonly = 1 READONLY
		TO %s
	`,
		profileName,
		config.MaxQueryExecutionTime,
		AnalyticsExecutionTimeMax,
		config.MaxQueryMemoryBytes,
		config.MaxQueryResultRows,
		AnalyticsResultBytesMax,
		AnalyticsASTDepthMax,
		AnalyticsASTElementsMax,
		config.Username,
	)
	err = c.Exec(ctx, createOrReplaceProfileSQL)
	if err != nil {
		return fmt.Errorf("failed to create/replace settings profile: %w", err)
	}

	logger.Info("successfully configured clickhouse user",
		"tables", len(config.AllowedTables),
		"max_queries_per_window", config.MaxQueriesPerWindow,
		"quota_duration_seconds", config.QuotaDurationSeconds,
	)

	return nil
}

// gatewayRequestColumns lists the columns of frontline_requests_raw_v1 that
// customers can read. It does not include instance_address, frontline_id, and
// platform. These three columns show the Unkey infrastructure and have no
// meaning for the workspace that made the traffic.
//
// workspace_id must stay in the list. The query parser injects
// `<table>.workspace_id = '<ws>'` into each query, and ClickHouse denies a query
// that names a column with no SELECT grant, even when the column appears only in
// WHERE. See injectWorkspaceFilterOnSelect in pkg/clickhouse/query-parser.
var gatewayRequestColumns = []string{
	"request_id",
	"time",
	"workspace_id",
	"project_id",
	"app_id",
	"environment_id",
	"deployment_id",
	"instance_id",
	"region",
	"method",
	"host",
	"path",
	"query_string",
	"query_params",
	"request_headers",
	"request_body",
	"response_status",
	"response_headers",
	"response_body",
	"user_agent",
	"ip_address",
	"total_latency",
	"instance_latency",
	"frontline_latency",
}

// DefaultAllowedTables returns the default list of tables for analytics access
func DefaultAllowedTables() []AllowedTable {
	return []AllowedTable{
		// Key verifications
		{Name: "default.key_verifications_raw_v2", Columns: nil},
		{Name: "default.key_verifications_per_minute_v3", Columns: nil},
		{Name: "default.key_verifications_per_hour_v3", Columns: nil},
		{Name: "default.key_verifications_per_day_v3", Columns: nil},
		{Name: "default.key_verifications_per_month_v3", Columns: nil},
		// Rate limits
		{Name: "default.ratelimits_raw_v2", Columns: nil},
		{Name: "default.ratelimits_per_minute_v2", Columns: nil},
		{Name: "default.ratelimits_per_hour_v2", Columns: nil},
		{Name: "default.ratelimits_per_day_v2", Columns: nil},
		{Name: "default.ratelimits_per_month_v2", Columns: nil},
		// Gateway requests
		{Name: "default.frontline_requests_raw_v1", Columns: gatewayRequestColumns},
	}
}
