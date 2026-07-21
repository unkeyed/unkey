package clickhouse

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	driver "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/unkeyed/unkey/pkg/logger"
)

type userExec func(context.Context, string, ...any) error

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

// UserConfig contains configuration for creating/updating a ClickHouse user
type UserConfig struct {
	WorkspaceID string
	Username    string
	Password    string

	// Tables to grant SELECT permission on
	AllowedTables []string

	// Quota settings (per window)
	QuotaDurationSeconds      int32
	MaxQueriesPerWindow       int32
	MaxExecutionTimePerWindow int32

	// Per-query limits (settings profile)
	MaxQueryExecutionTime int32
	MaxQueryMemoryBytes   int64
	MaxQueryResultRows    int32

	// Data retention (in days) - read from quotas table
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

	// Validate all table names
	for _, table := range config.AllowedTables {
		if !validTableName.MatchString(table) {
			return fmt.Errorf("invalid table name: must be in format 'database.table' with alphanumeric characters and underscores only, got %q", table)
		}
	}

	return nil
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
	return configureUserWithExec(ctx, config, c.Exec)
}

// RevokeUserAccess closes an existing user's direct grants before durable state changes.
func (c *Client) RevokeUserAccess(ctx context.Context, username string) error {
	if !validIdentifier.MatchString(username) {
		return fmt.Errorf("invalid username: must contain only alphanumeric characters and underscores, got %q", username)
	}
	if err := c.Exec(ctx, fmt.Sprintf("REVOKE ALL ON *.* FROM %s", username)); err != nil {
		return fmt.Errorf("failed to revoke permissions: %w", err)
	}
	return nil
}

func configureUserWithExec(ctx context.Context, config UserConfig, exec userExec) error {

	// Validate all identifiers to prevent SQL injection
	if err := validateIdentifiers(config); err != nil {
		return fmt.Errorf("identifier validation failed: %w", err)
	}

	// Create or alter ClickHouse user
	logger.Info("creating/updating clickhouse user")
	createUserSQL := fmt.Sprintf("CREATE USER IF NOT EXISTS %s IDENTIFIED WITH sha256_password BY {password:String}", config.Username)
	err := exec(ctx, createUserSQL, driver.Named("password", config.Password))
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	// Revoke all permissions
	logger.Info("revoking all permissions")
	revokeSQL := fmt.Sprintf("REVOKE ALL ON *.* FROM %s", config.Username)
	err = exec(ctx, revokeSQL)
	if err != nil {
		return fmt.Errorf("failed to revoke permissions: %w", err)
	}

	// Create row-level security (RLS) policies
	policyName := fmt.Sprintf("workspace_%s_rls", config.WorkspaceID)
	for _, table := range config.AllowedTables {
		logger.Debug("creating row policy", "table", table, "policy", policyName, "retention_days", config.RetentionDays)

		// Build time retention filter based on table type and configured retention period
		timeFilter := getTimeRetentionFilter(table, config.RetentionDays)

		createPolicySQL := fmt.Sprintf(
			"CREATE ROW POLICY OR REPLACE %s ON %s FOR SELECT USING workspace_id = '%s' AND (%s) TO %s",
			policyName, table, config.WorkspaceID, timeFilter, config.Username,
		)
		err = exec(ctx, createPolicySQL)
		if err != nil {
			return fmt.Errorf("failed to create row policy on %s: %w", table, err)
		}
	}

	// Create or replace settings profile
	profileName := fmt.Sprintf("workspace_%s_profile", config.WorkspaceID)
	logger.Info("creating/updating settings profile", "name", profileName)

	createOrReplaceProfileSQL := fmt.Sprintf(`
		CREATE SETTINGS PROFILE OR REPLACE %s SETTINGS
			max_execution_time = %d CONST,
			max_memory_usage = %d CONST,
			max_result_rows = %d CONST,
			readonly = 2 CONST
		TO %s
	`,
		profileName,
		config.MaxQueryExecutionTime,
		config.MaxQueryMemoryBytes,
		config.MaxQueryResultRows,
		config.Username,
	)
	err = exec(ctx, createOrReplaceProfileSQL)
	if err != nil {
		return fmt.Errorf("failed to create/replace settings profile: %w", err)
	}

	// Create or replace quota
	quotaName := fmt.Sprintf("workspace_%s_quota", config.WorkspaceID)
	logger.Info("creating/updating quota", "name", quotaName)

	createOrReplaceQuotaSQL := fmt.Sprintf(`
		CREATE QUOTA OR REPLACE %s
		FOR INTERVAL %d SECOND
			MAX queries = %d,
			MAX execution_time = %d
		TO %s
	`,
		quotaName,
		config.QuotaDurationSeconds,
		config.MaxQueriesPerWindow,
		config.MaxExecutionTimePerWindow,
		config.Username,
	)
	err = exec(ctx, createOrReplaceQuotaSQL)
	if err != nil {
		return fmt.Errorf("failed to create/replace quota: %w", err)
	}

	// A single final GRANT prevents any table from becoming readable while
	// another table is still missing its policy or resource constraints.
	if len(config.AllowedTables) > 0 {
		grants := make([]string, 0, len(config.AllowedTables))
		for _, table := range config.AllowedTables {
			grants = append(grants, "SELECT ON "+table)
		}
		grantSQL := fmt.Sprintf("GRANT %s TO %s", strings.Join(grants, ", "), config.Username)
		if err = exec(ctx, grantSQL); err != nil {
			return fmt.Errorf("failed to grant SELECT: %w", err)
		}
	}

	logger.Info("successfully configured clickhouse user",
		"tables", len(config.AllowedTables),
		"max_queries_per_window", config.MaxQueriesPerWindow,
		"quota_duration_seconds", config.QuotaDurationSeconds,
	)

	return nil
}

// DefaultAllowedTables returns the default list of tables for analytics access
func DefaultAllowedTables() []string {
	return []string{
		// Key verifications
		"default.key_verifications_raw_v2",
		"default.key_verifications_per_minute_v3",
		"default.key_verifications_per_hour_v3",
		"default.key_verifications_per_day_v3",
		"default.key_verifications_per_month_v3",
	}
}
