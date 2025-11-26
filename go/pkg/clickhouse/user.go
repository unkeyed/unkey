package clickhouse

import (
	"context"
	"fmt"
	"regexp"

	driver "github.com/ClickHouse/clickhouse-go/v2"
)

var (
	// validIdentifier matches safe ClickHouse identifiers (usernames, policy names, quota names, profile names)
	// Allows alphanumeric characters and underscores only
	validIdentifier = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

	// validTableName matches safe ClickHouse table names in database.table format
	// Allows alphanumeric characters and underscores in both database and table parts
	validTableName = regexp.MustCompile(`^[a-zA-Z0-9_]+\.[a-zA-Z0-9_]+$`)
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

// ConfigureUser creates or updates a ClickHouse user with all necessary permissions, quotas, and settings.
// This is idempotent - it can be run multiple times to update settings.
func (c *clickhouse) ConfigureUser(ctx context.Context, config UserConfig) error {
	logger := c.logger.With("workspace_id", config.WorkspaceID, "username", config.Username)

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
		logger.Debug("granting SELECT permission", "table", table)
		grantSQL := fmt.Sprintf("GRANT SELECT ON %s TO %s", table, config.Username)
		err = c.Exec(ctx, grantSQL)
		if err != nil {
			return fmt.Errorf("failed to grant SELECT on %s: %w", table, err)
		}
	}

	// Create row-level security (RLS) policies
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
			max_execution_time = %d,
			max_memory_usage = %d,
			max_result_rows = %d,
			readonly = 2
		TO %s
	`,
		profileName,
		config.MaxQueryExecutionTime,
		config.MaxQueryMemoryBytes,
		config.MaxQueryResultRows,
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
		"default.key_verifications_per_minute_v3",
		"default.key_verifications_per_hour_v2",
		"default.key_verifications_per_hour_v3",
		"default.key_verifications_per_day_v2",
		"default.key_verifications_per_day_v3",
		"default.key_verifications_per_month_v2",
		"default.key_verifications_per_month_v3",
	}
}
