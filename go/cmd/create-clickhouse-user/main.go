package createclickhouseuser

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"

	vaultv1 "github.com/unkeyed/unkey/go/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/go/pkg/cli"
	"github.com/unkeyed/unkey/go/pkg/clickhouse"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/vault"
	"github.com/unkeyed/unkey/go/pkg/vault/storage"
)

var Cmd = &cli.Command{
	Name:  "create-clickhouse-user",
	Usage: "Create or update ClickHouse user with quotas and permissions",
	Description: `Create or update a ClickHouse user for a workspace with resource quotas and table permissions.

This command:
1. Generates a secure password (or reuses existing)
2. Encrypts and stores credentials in MySQL
3. Creates/alters ClickHouse user
4. Revokes all permissions
5. Grants SELECT on specified tables
6. Creates row-level security policies
7. Creates/updates quota
8. Creates/updates settings profile

The script is idempotent - it can be run multiple times to update quotas without regenerating passwords.

By default, grants SELECT on only key verification v2 analytics tables:
  - Key verifications (raw + per minute/hour/day/month)

EXAMPLES:
unkey create-clickhouse-user --workspace-id ws_123
unkey create-clickhouse-user --workspace-id ws_123 --username custom_user --max-queries-per-window 5000`,
	Flags: []cli.Flag{
		cli.String("workspace-id", "Workspace ID", cli.Required()),
		cli.String("username", "ClickHouse username (default: workspace_id)"),
		cli.String("database-primary", "MySQL database DSN", cli.EnvVar("UNKEY_DATABASE_PRIMARY"), cli.Required()),
		cli.String("clickhouse-url", "ClickHouse URL", cli.EnvVar("CLICKHOUSE_URL"), cli.Required()),
		cli.StringSlice("vault-master-keys", "Vault master key for encryption", cli.EnvVar("UNKEY_VAULT_MASTER_KEY"), cli.Required()),
		cli.String("vault-s3-url", "Vault S3 URL", cli.EnvVar("UNKEY_VAULT_S3_URL"), cli.Required()),
		cli.String("vault-s3-bucket", "Vault S3 bucket", cli.EnvVar("UNKEY_VAULT_S3_BUCKET"), cli.Required()),
		cli.String("vault-s3-access-key-id", "Vault S3 access key ID", cli.EnvVar("UNKEY_VAULT_S3_ACCESS_KEY_ID"), cli.Required()),
		cli.String("vault-s3-access-key-secret", "Vault S3 access key secret", cli.EnvVar("UNKEY_VAULT_S3_ACCESS_KEY_SECRET"), cli.Required()),

		// Quota overrides (optional - uses schema defaults if not set)
		cli.Int("quota-duration-seconds", "Quota window duration in seconds", cli.Default(3_600)),
		cli.Int("max-queries-per-window", "Max queries per window", cli.Default(1_000)),
		cli.Int("max-execution-time-per-window", "Max execution time per window in seconds", cli.Default(1_800)),
		cli.Int("max-query-execution-time", "Max single query execution time in seconds", cli.Default(30)),
		cli.Int64("max-query-memory-bytes", "Max memory per query in bytes", cli.Default(int64(1_000_000_000))),
		cli.Int("max-query-result-rows", "Max result rows per query", cli.Default(10_000_000)),
	},
	Action: run,
}

var allowedTables = []string{
	// Key verifications
	//
	"default.key_verifications_raw_v2",
	"default.key_verifications_per_minute_v2",
	"default.key_verifications_per_minute_v3",
	"default.key_verifications_per_hour_v2",
	"default.key_verifications_per_hour_v3",
	"default.key_verifications_per_day_v2",
	"default.key_verifications_per_day_v3",
	"default.key_verifications_per_month_v2",
	"default.key_verifications_per_month_v3",

	// === Not used ATM ===

	// // Ratelimits
	// "default.ratelimits_raw_v2",
	// "default.ratelimits_per_minute_v2",
	// "default.ratelimits_per_hour_v2",
	// "default.ratelimits_per_day_v2",
	// "default.ratelimits_per_month_v2",
	// "default.ratelimits_last_used_v2",

	// // API requests
	// "default.api_requests_raw_v2",
	// "default.api_requests_per_minute_v2",
	// "default.api_requests_per_hour_v2",
	// "default.api_requests_per_day_v2",
	// "default.api_requests_per_month_v2",
}

func run(ctx context.Context, cmd *cli.Command) error {
	logger := logging.New()

	workspaceID := cmd.RequireString("workspace-id")
	username := cmd.String("username")
	if username == "" {
		username = workspaceID
	}

	// Connect to MySQL
	database, err := db.New(db.Config{
		PrimaryDSN:  cmd.RequireString("database-primary"),
		ReadOnlyDSN: "",
		Logger:      logger,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Connect to ClickHouse
	ch, err := clickhouse.New(clickhouse.Config{
		URL:    cmd.RequireString("clickhouse-url"),
		Logger: logger,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to clickhouse: %w", err)
	}

	// Initialize Vault storage
	vaultStorage, err := storage.NewS3(storage.S3Config{
		Logger:            logger,
		S3URL:             cmd.RequireString("vault-s3-url"),
		S3Bucket:          cmd.RequireString("vault-s3-bucket"),
		S3AccessKeyID:     cmd.RequireString("vault-s3-access-key-id"),
		S3AccessKeySecret: cmd.RequireString("vault-s3-access-key-secret"),
	})
	if err != nil {
		return fmt.Errorf("failed to initialize vault storage: %w", err)
	}

	// Initialize Vault for encryption
	v, err := vault.New(vault.Config{
		Logger:     logger,
		Storage:    vaultStorage,
		MasterKeys: cmd.RequireStringSlice("vault-master-keys"),
	})
	if err != nil {
		return fmt.Errorf("failed to initialize vault: %w", err)
	}

	clk := clock.New()
	now := clk.Now().UnixMilli()

	// Check if user already exists (also fetches quota with retention days)
	existing, err := db.Query.FindClickhouseWorkspaceSettingsByWorkspaceID(ctx, database.RO(), workspaceID)
	var password string
	var passwordEncrypted string
	var retentionDays int32

	if err != nil {
		if !db.IsNotFound(err) {
			return fmt.Errorf("failed to check existing user: %w", err)
		}

		// User doesn't exist - need to fetch quota separately and generate new password
		logger.Info("creating new user", "workspace_id", workspaceID, "username", username)

		// Fetch workspace quota for retention days
		quota, err := db.Query.FindQuotaByWorkspaceID(ctx, database.RO(), workspaceID)
		if err != nil {
			return fmt.Errorf("failed to fetch workspace quota: %w", err)
		}
		retentionDays = quota.LogsRetentionDays

		password, err = generateSecurePassword(64)
		if err != nil {
			return fmt.Errorf("failed to generate password: %w", err)
		}

		// Encrypt password
		var encRes *vaultv1.EncryptResponse
		encRes, err = v.Encrypt(ctx, &vaultv1.EncryptRequest{
			Keyring: workspaceID,
			Data:    password,
		})
		if err != nil {
			return fmt.Errorf("failed to encrypt password: %w", err)
		}
		passwordEncrypted = encRes.GetEncrypted()

		// Insert into MySQL
		err = db.Query.InsertClickhouseWorkspaceSettings(ctx, database.RW(), db.InsertClickhouseWorkspaceSettingsParams{
			WorkspaceID:               workspaceID,
			Username:                  username,
			PasswordEncrypted:         passwordEncrypted,
			QuotaDurationSeconds:      int32(cmd.Int("quota-duration-seconds")),
			MaxQueriesPerWindow:       int32(cmd.Int("max-queries-per-window")),
			MaxExecutionTimePerWindow: int32(cmd.Int("max-execution-time-per-window")),
			MaxQueryExecutionTime:     int32(cmd.Int("max-query-execution-time")),
			MaxQueryMemoryBytes:       cmd.Int64("max-query-memory-bytes"),
			MaxQueryResultRows:        int32(cmd.Int("max-query-result-rows")),
			CreatedAt:                 now,
			UpdatedAt:                 sql.NullInt64{Valid: true, Int64: now},
		})
		if err != nil {
			return fmt.Errorf("failed to insert settings: %w", err)
		}

		logger.Info("stored credentials in database")
	} else {
		// User exists - update quotas only (preserve password, get retention from quota)
		logger.Info("updating existing user quotas", "workspace_id", workspaceID, "username", existing.ClickhouseWorkspaceSetting.Username)
		username = existing.ClickhouseWorkspaceSetting.Username
		retentionDays = existing.Quotas.LogsRetentionDays
		decrypted, err := v.Decrypt(ctx, &vaultv1.DecryptRequest{
			Keyring:   workspaceID,
			Encrypted: existing.ClickhouseWorkspaceSetting.PasswordEncrypted,
		})
		if err != nil {
			return fmt.Errorf("failed to decrypt password: %w", err)
		}
		password = decrypted.GetPlaintext()

		// Update limits
		err = db.Query.UpdateClickhouseWorkspaceSettingsLimits(ctx, database.RW(), db.UpdateClickhouseWorkspaceSettingsLimitsParams{
			WorkspaceID:               workspaceID,
			QuotaDurationSeconds:      int32(cmd.Int("quota-duration-seconds")),
			MaxQueriesPerWindow:       int32(cmd.Int("max-queries-per-window")),
			MaxExecutionTimePerWindow: int32(cmd.Int("max-execution-time-per-window")),
			MaxQueryExecutionTime:     int32(cmd.Int("max-query-execution-time")),
			MaxQueryMemoryBytes:       cmd.Int64("max-query-memory-bytes"),
			MaxQueryResultRows:        int32(cmd.Int("max-query-result-rows")),
			UpdatedAt:                 sql.NullInt64{Valid: true, Int64: now},
		})
		if err != nil {
			return fmt.Errorf("failed to update settings: %w", err)
		}

		logger.Info("updated quotas in database")
	}

	// Configure ClickHouse user with permissions, quotas, and settings
	err = ch.ConfigureUser(ctx, clickhouse.UserConfig{
		WorkspaceID:               workspaceID,
		Username:                  username,
		Password:                  password,
		AllowedTables:             allowedTables,
		QuotaDurationSeconds:      int32(cmd.Int("quota-duration-seconds")),
		MaxQueriesPerWindow:       int32(cmd.Int("max-queries-per-window")),
		MaxExecutionTimePerWindow: int32(cmd.Int("max-execution-time-per-window")),
		MaxQueryExecutionTime:     int32(cmd.Int("max-query-execution-time")),
		MaxQueryMemoryBytes:       cmd.Int64("max-query-memory-bytes"),
		MaxQueryResultRows:        int32(cmd.Int("max-query-result-rows")),
		RetentionDays:             retentionDays,
	})
	if err != nil {
		return fmt.Errorf("failed to configure clickhouse user: %w", err)
	}

	return nil
}

func generateSecurePassword(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(bytes)[:length], nil
}
