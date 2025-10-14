package clickhouse

import (
	"context"
	"fmt"

	driver "github.com/ClickHouse/clickhouse-go/v2"
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
	MaxRowsToRead         int64
}

// ConfigureUser creates or updates a ClickHouse user with all necessary permissions, quotas, and settings.
// This is idempotent - it can be run multiple times to update settings.
//
// Steps:
// 1. Create/alter user with password
// 2. Revoke all existing permissions
// 3. Grant SELECT on specified tables
// 4. Create row-level security policies (filters by workspace_id)
// 5. Create/update quota (max queries and execution time per window)
// 6. Create/update settings profile (per-query limits)
func (c *clickhouse) ConfigureUser(ctx context.Context, config UserConfig) error {
	logger := c.logger.With("workspace_id", config.WorkspaceID, "username", config.Username)

	// 1. Create or alter ClickHouse user
	logger.Info("creating/updating clickhouse user")
	createUserSQL := fmt.Sprintf("CREATE USER IF NOT EXISTS %s IDENTIFIED WITH sha256_password BY {password:String}", config.Username)
	err := c.Exec(ctx, createUserSQL, driver.Named("password", config.Password))
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	// 2. Revoke all permissions
	logger.Info("revoking all permissions")
	revokeSQL := fmt.Sprintf("REVOKE ALL ON *.* FROM %s", config.Username)
	err = c.Exec(ctx, revokeSQL)
	if err != nil {
		logger.Warn("failed to revoke permissions (user may be new)", "error", err)
	}

	// 3. Grant SELECT on specified tables
	for _, table := range config.AllowedTables {
		logger.Debug("granting SELECT permission", "table", table)
		grantSQL := fmt.Sprintf("GRANT SELECT ON %s TO %s", table, config.Username)
		err = c.Exec(ctx, grantSQL)
		if err != nil {
			return fmt.Errorf("failed to grant SELECT on %s: %w", table, err)
		}
	}

	// 4. Create row-level security (RLS) policies
	policyName := fmt.Sprintf("workspace_%s_rls", config.WorkspaceID)
	for _, table := range config.AllowedTables {
		logger.Debug("creating row policy", "table", table, "policy", policyName)

		createPolicySQL := fmt.Sprintf(
			"CREATE ROW POLICY OR REPLACE %s ON %s FOR SELECT USING workspace_id = '%s' TO %s",
			policyName, table, config.WorkspaceID, config.Username,
		)
		err = c.Exec(ctx, createPolicySQL)
		if err != nil {
			return fmt.Errorf("failed to create row policy on %s: %w", table, err)
		}
	}

	// 5. Create or replace quota
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

	// 6. Create or replace settings profile
	profileName := fmt.Sprintf("workspace_%s_profile", config.WorkspaceID)
	logger.Info("creating/updating settings profile", "name", profileName)

	createOrReplaceProfileSQL := fmt.Sprintf(`
		CREATE SETTINGS PROFILE OR REPLACE %s SETTINGS
			max_execution_time = %d,
			max_memory_usage = %d,
			max_result_rows = %d,
			max_rows_to_read = %d,
			readonly = 2
		TO %s
	`,
		profileName,
		config.MaxQueryExecutionTime,
		config.MaxQueryMemoryBytes,
		config.MaxQueryResultRows,
		config.MaxRowsToRead,
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

// DefaultAllowedTables returns the default list of tables for analytics access
func DefaultAllowedTables() []string {
	return []string{
		// Key verifications
		"default.key_verifications_raw_v2",
		"default.key_verifications_per_minute_v2",
		"default.key_verifications_per_hour_v2",
		"default.key_verifications_per_day_v2",
		"default.key_verifications_per_month_v2",
	}
}
