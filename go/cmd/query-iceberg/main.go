package queryiceberg

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	vaultv1 "github.com/unkeyed/unkey/go/gen/proto/vault/v1"
	icebergstorage "github.com/unkeyed/unkey/go/pkg/analytics/storage/iceberg"
	"github.com/unkeyed/unkey/go/pkg/cli"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/vault"
	vaultstorage "github.com/unkeyed/unkey/go/pkg/vault/storage"

	_ "github.com/marcboeker/go-duckdb"
)

var Cmd = &cli.Command{
	Name:  "query-iceberg",
	Usage: "Query Iceberg analytics data from R2 using DuckDB",
	Flags: []cli.Flag{
		cli.String("database-primary", "Primary database connection string",
			cli.Required(), cli.EnvVar("UNKEY_DATABASE_PRIMARY")),
		cli.String("workspace-id", "Workspace ID to query",
			cli.Required()),
		cli.String("query", "SQL query to execute",
			cli.Required()),
		cli.StringSlice("vault-master-keys", "Vault master keys for decryption",
			cli.Required(), cli.EnvVar("VAULT_MASTER_KEYS")),
		cli.String("vault-s3-url", "Vault S3 URL",
			cli.Required(), cli.EnvVar("VAULT_S3_URL")),
		cli.String("vault-s3-bucket", "Vault S3 bucket",
			cli.Required(), cli.EnvVar("VAULT_S3_BUCKET")),
		cli.String("vault-s3-access-key-id", "Vault S3 access key ID",
			cli.Required(), cli.EnvVar("VAULT_S3_ACCESS_KEY_ID")),
		cli.String("vault-s3-access-key-secret", "Vault S3 secret access key",
			cli.Required(), cli.EnvVar("VAULT_S3_ACCESS_KEY_SECRET")),
	},
	Action: action,
}

func action(ctx context.Context, cmd *cli.Command) error {
	logger := logging.New()

	// Connect to database
	database, err := db.New(db.Config{
		PrimaryDSN: cmd.String("database-primary"),
		Logger:     logger,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer database.Close()

	// Initialize vault service for decryption
	vaultStorage, err := vaultstorage.NewS3(vaultstorage.S3Config{
		Logger:            logger,
		S3URL:             cmd.String("vault-s3-url"),
		S3Bucket:          cmd.String("vault-s3-bucket"),
		S3AccessKeyID:     cmd.String("vault-s3-access-key-id"),
		S3AccessKeySecret: cmd.String("vault-s3-access-key-secret"),
	})
	if err != nil {
		return fmt.Errorf("failed to create vault storage: %w", err)
	}

	vaultService, err := vault.New(vault.Config{
		Logger:     logger,
		Storage:    vaultStorage,
		MasterKeys: cmd.StringSlice("vault-master-keys"),
	})
	if err != nil {
		return fmt.Errorf("failed to initialize vault: %w", err)
	}

	// Fetch analytics config for workspace
	workspaceID := cmd.String("workspace-id")
	queries := &db.Queries{}
	configRow, err := queries.FindAnalyticsConfigByWorkspaceID(ctx, database.RO(), workspaceID)
	if err != nil {
		return fmt.Errorf("failed to fetch analytics config: %w", err)
	}

	var config icebergstorage.WorkspaceConfig
	if err := json.Unmarshal(configRow.Config, &config); err != nil {
		return fmt.Errorf("failed to parse analytics config: %w", err)
	}

	// Decrypt credentials
	if config.EncryptedAccessKeyId != "" {
		resp, err := vaultService.Decrypt(ctx, &vaultv1.DecryptRequest{
			Keyring:   workspaceID,
			Encrypted: config.EncryptedAccessKeyId,
		})
		if err != nil {
			return fmt.Errorf("failed to decrypt access key ID: %w", err)
		}
		config.AccessKeyId = resp.Plaintext
	}

	if config.EncryptedSecretAccessKey != "" {
		resp, err := vaultService.Decrypt(ctx, &vaultv1.DecryptRequest{
			Keyring:   workspaceID,
			Encrypted: config.EncryptedSecretAccessKey,
		})
		if err != nil {
			return fmt.Errorf("failed to decrypt secret access key: %w", err)
		}
		config.SecretAccessKey = resp.Plaintext
	}

	if config.EncryptedCatalogToken != "" {
		resp, err := vaultService.Decrypt(ctx, &vaultv1.DecryptRequest{
			Keyring:   workspaceID,
			Encrypted: config.EncryptedCatalogToken,
		})
		if err != nil {
			return fmt.Errorf("failed to decrypt catalog token: %w", err)
		}
		config.CatalogToken = resp.Plaintext
	}

	logger.Info("credentials decrypted",
		"bucket", config.Bucket,
		"endpoint", config.Endpoint,
		"catalogEndpoint", config.CatalogEndpoint,
	)

	// Connect to DuckDB
	duckDB, err := sql.Open("duckdb", "")
	if err != nil {
		return fmt.Errorf("failed to open DuckDB: %w", err)
	}
	defer duckDB.Close()

	// Install and load httpfs first (required for remote access)
	logger.Info("installing httpfs extension")
	if _, err := duckDB.ExecContext(ctx, "INSTALL httpfs"); err != nil {
		return fmt.Errorf("failed to install httpfs extension: %w", err)
	}
	if _, err := duckDB.ExecContext(ctx, "LOAD httpfs"); err != nil {
		return fmt.Errorf("failed to load httpfs extension: %w", err)
	}

	// Install and load iceberg extension
	logger.Info("installing iceberg extension")
	if _, err := duckDB.ExecContext(ctx, "INSTALL iceberg"); err != nil {
		return fmt.Errorf("failed to install iceberg extension: %w", err)
	}
	if _, err := duckDB.ExecContext(ctx, "LOAD iceberg"); err != nil {
		return fmt.Errorf("failed to load iceberg extension: %w", err)
	}

	if _, err := duckDB.ExecContext(ctx, "UPDATE EXTENSIONS"); err != nil {
		return fmt.Errorf("failed to load iceberg extension: %w", err)
	}

	// Create R2 Iceberg secret (handles both catalog and data access)
	logger.Info("creating R2 Iceberg secret")
	createR2SecretSQL := fmt.Sprintf(`
		CREATE SECRET r2_secret (
			TYPE iceberg,
			TOKEN '%s'
		)
	`, config.CatalogToken)

	if _, err := duckDB.ExecContext(ctx, createR2SecretSQL); err != nil {
		return fmt.Errorf("failed to create R2 Iceberg secret: %w", err)
	}

	// Attach R2 catalog
	logger.Info("attaching R2 catalog")
	attachCatalogSQL := fmt.Sprintf(`
		ATTACH '%s' AS r2_catalog (
			TYPE iceberg,
			ENDPOINT '%s'
		)
	`, config.Bucket, config.CatalogEndpoint)

	if _, err := duckDB.ExecContext(ctx, attachCatalogSQL); err != nil {
		return fmt.Errorf("failed to attach R2 catalog: %w", err)
	}

	logger.Info("R2 catalog attached successfully",
		"warehouse", config.Bucket,
		"endpoint", config.CatalogEndpoint,
	)

	// List available tables
	logger.Info("listing available tables")
	showTablesSQL := "SHOW ALL TABLES"
	tablesRows, err := duckDB.QueryContext(ctx, showTablesSQL)
	if err != nil {
		logger.Warn("failed to list tables", "error", err.Error())
	} else {
		defer tablesRows.Close()
		logger.Info("available tables:")
		for tablesRows.Next() {
			var database, schema, tableName, columnNames, columnTypes, temporary string
			if err := tablesRows.Scan(&database, &schema, &tableName, &columnNames, &columnTypes, &temporary); err == nil {
				logger.Info("  table found", "database", database, "schema", schema, "name", tableName)
			}
		}
	}

	// Execute user query
	query := cmd.String("query")
	logger.Info("executing query", "query", query)

	rows, err := duckDB.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get columns: %w", err)
	}

	// Print header
	for i, col := range columns {
		if i > 0 {
			fmt.Print("\t")
		}
		fmt.Print(col)
	}
	fmt.Println()

	// Print separator
	for i := range columns {
		if i > 0 {
			fmt.Print("\t")
		}
		fmt.Print("---")
	}
	fmt.Println()

	// Print rows
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	rowCount := 0
	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}

		for i, val := range values {
			if i > 0 {
				fmt.Print("\t")
			}
			if val == nil {
				fmt.Print("NULL")
			} else {
				fmt.Print(val)
			}
		}
		fmt.Println()
		rowCount++
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating rows: %w", err)
	}

	logger.Info("query completed", "rows", rowCount)
	return nil
}

func extractAccountID(catalogEndpoint string) string {
	// Extract account ID from catalog URL
	// Format: https://catalog.cloudflarestorage.com/{account_id}/{bucket}
	parts := strings.Split(catalogEndpoint, "/")
	if len(parts) >= 4 {
		return parts[3]
	}
	return ""
}
