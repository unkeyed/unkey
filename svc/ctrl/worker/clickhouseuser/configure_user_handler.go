package clickhouseuser

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"time"

	"connectrpc.com/connect"
	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/db"
)

// Default quota values matching the original script
const (
	defaultQuotaDurationSeconds      = 3600       // 1 hour
	defaultMaxQueriesPerWindow       = 1000       // queries
	defaultMaxExecutionTimePerWindow = 1800       // 30 minutes
	defaultMaxQueryExecutionTime     = 30         // seconds
	defaultMaxQueryMemoryBytes       = 1000000000 // 1 GB
	defaultMaxQueryResultRows        = 10000000   // 10 million rows
	passwordLength                   = 64         // characters
)

// ConfigureUser creates or updates a ClickHouse user for a workspace.
//
// This is a Restate virtual object handler keyed by workspace_id, ensuring only one
// user provisioning runs per workspace at any time. The workflow consists of durable
// steps that survive process restarts: checking existing settings, generating/retrieving
// password, encrypting/decrypting via vault, storing in MySQL, and configuring ClickHouse.
//
// For new users, the workflow generates a secure password, encrypts it via the vault API,
// stores it in MySQL, and creates the ClickHouse user with permissions and quotas.
//
// For existing users, the workflow retrieves and decrypts the existing password,
// updates quota settings in MySQL, and reapplies the ClickHouse configuration to ensure
// consistency (quotas, row policies, settings profile).
//
// Security: Plaintext passwords are never journaled by Restate. Password generation,
// encryption, and decryption happen within single restate.Run blocks that only return
// non-sensitive data (encrypted passwords or success indicators).
//
// Returns a response with Status "success" and the username on success, or Status
// "failed" with an error message on failure.
func (s *Service) ConfigureUser(
	ctx restate.ObjectContext,
	req *hydrav1.ConfigureUserRequest,
) (*hydrav1.ConfigureUserResponse, error) {
	workspaceID := req.GetWorkspaceId()
	s.logger.Info("starting clickhouse user configuration", "workspace_id", workspaceID)

	// Resolve username (default to workspace_id if not provided)
	username := req.GetUsername()
	if username == "" {
		username = workspaceID
	}

	// Resolve quota settings with defaults
	quotaSettings := resolveQuotaSettings(req)

	// Step 1: Check if user already exists in MySQL
	existing, err := restate.Run(ctx, func(stepCtx restate.RunContext) (db.FindClickhouseWorkspaceSettingsByWorkspaceIDRow, error) {
		return db.Query.FindClickhouseWorkspaceSettingsByWorkspaceID(stepCtx, s.db.RO(), workspaceID)
	}, restate.WithName("check existing user"))

	isNewUser := err != nil && db.IsNotFound(err)
	if err != nil && !isNewUser {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}

	var encryptedPassword string
	var retentionDays int32

	if isNewUser {
		// New user flow
		s.logger.Info("creating new clickhouse user", "workspace_id", workspaceID, "username", username)

		// Step 2a: Fetch quota for retention days
		quota, err := restate.Run(ctx, func(stepCtx restate.RunContext) (db.Quotum, error) {
			return db.Query.FindQuotaByWorkspaceID(stepCtx, s.db.RO(), workspaceID)
		}, restate.WithName("fetch quota"))
		if err != nil {
			return nil, fmt.Errorf("failed to fetch workspace quota: %w", err)
		}
		retentionDays = quota.LogsRetentionDays

		// Step 2b: Generate password, encrypt it, and store in MySQL
		// All in one step so plaintext password is never journaled by Restate.
		//
		// This step is idempotent: if a previous attempt inserted to MySQL but crashed
		// before Restate journaled the result, we detect the existing record and return
		// its encrypted password instead of generating a new one.
		encryptedPassword, err = restate.Run(ctx, func(stepCtx restate.RunContext) (string, error) {
			// Generate secure password (ephemeral - not journaled)
			password, genErr := generateSecurePassword(passwordLength)
			if genErr != nil {
				return "", fmt.Errorf("failed to generate password: %w", genErr)
			}

			// Encrypt password via vault API
			resp, encErr := s.vault.Encrypt(stepCtx, connect.NewRequest(&vaultv1.EncryptRequest{
				Keyring: workspaceID,
				Data:    password,
			}))
			if encErr != nil {
				return "", fmt.Errorf("failed to encrypt password: %w", encErr)
			}
			encrypted := resp.Msg.GetEncrypted()

			// Insert settings to MySQL
			now := time.Now().UnixMilli()
			insertErr := db.Query.InsertClickhouseWorkspaceSettings(stepCtx, s.db.RW(), db.InsertClickhouseWorkspaceSettingsParams{
				WorkspaceID:               workspaceID,
				Username:                  username,
				PasswordEncrypted:         encrypted,
				QuotaDurationSeconds:      quotaSettings.quotaDurationSeconds,
				MaxQueriesPerWindow:       quotaSettings.maxQueriesPerWindow,
				MaxExecutionTimePerWindow: quotaSettings.maxExecutionTimePerWindow,
				MaxQueryExecutionTime:     quotaSettings.maxQueryExecutionTime,
				MaxQueryMemoryBytes:       quotaSettings.maxQueryMemoryBytes,
				MaxQueryResultRows:        quotaSettings.maxQueryResultRows,
				CreatedAt:                 now,
				UpdatedAt:                 sql.NullInt64{Valid: true, Int64: now},
			})
			if insertErr != nil {
				// Check if this is a duplicate key error (previous attempt succeeded but wasn't journaled)
				// If so, fetch the existing record and return its encrypted password for consistency
				if db.IsDuplicateKeyError(insertErr) {
					existing, fetchErr := db.Query.FindClickhouseWorkspaceSettingsByWorkspaceID(stepCtx, s.db.RO(), workspaceID)
					if fetchErr != nil {
						return "", fmt.Errorf("failed to fetch existing settings after duplicate: %w", fetchErr)
					}
					return existing.ClickhouseWorkspaceSetting.PasswordEncrypted, nil
				}
				return "", fmt.Errorf("failed to insert settings: %w", insertErr)
			}

			// Only return encrypted password - plaintext stays ephemeral
			return encrypted, nil
		}, restate.WithName("generate and store credentials"))
		if err != nil {
			return nil, err
		}

		s.logger.Info("stored credentials in database", "workspace_id", workspaceID)

	} else {
		// Existing user flow - update quotas only, preserve password
		s.logger.Info("updating existing clickhouse user", "workspace_id", workspaceID, "username", existing.ClickhouseWorkspaceSetting.Username)
		username = existing.ClickhouseWorkspaceSetting.Username
		retentionDays = existing.Quotas.LogsRetentionDays
		encryptedPassword = existing.ClickhouseWorkspaceSetting.PasswordEncrypted

		// Step 3: Update limits in MySQL
		now := time.Now().UnixMilli()
		_, err = restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
			return restate.Void{}, db.Query.UpdateClickhouseWorkspaceSettingsLimits(stepCtx, s.db.RW(), db.UpdateClickhouseWorkspaceSettingsLimitsParams{
				WorkspaceID:               workspaceID,
				QuotaDurationSeconds:      quotaSettings.quotaDurationSeconds,
				MaxQueriesPerWindow:       quotaSettings.maxQueriesPerWindow,
				MaxExecutionTimePerWindow: quotaSettings.maxExecutionTimePerWindow,
				MaxQueryExecutionTime:     quotaSettings.maxQueryExecutionTime,
				MaxQueryMemoryBytes:       quotaSettings.maxQueryMemoryBytes,
				MaxQueryResultRows:        quotaSettings.maxQueryResultRows,
				UpdatedAt:                 sql.NullInt64{Valid: true, Int64: now},
			})
		}, restate.WithName("update limits"))
		if err != nil {
			return nil, fmt.Errorf("failed to update settings: %w", err)
		}

		s.logger.Info("updated quotas in database", "workspace_id", workspaceID)
	}

	// Step 4: Configure ClickHouse user with permissions, quotas, and settings
	// Decrypt password inside this step so plaintext is never journaled
	_, err = restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		// Decrypt password (ephemeral - not journaled)
		resp, decErr := s.vault.Decrypt(stepCtx, connect.NewRequest(&vaultv1.DecryptRequest{
			Keyring:   workspaceID,
			Encrypted: encryptedPassword,
		}))
		if decErr != nil {
			return restate.Void{}, fmt.Errorf("failed to decrypt password: %w", decErr)
		}
		password := resp.Msg.GetPlaintext()

		// Configure ClickHouse user
		return restate.Void{}, s.clickhouse.ConfigureUser(stepCtx, clickhouse.UserConfig{
			WorkspaceID:               workspaceID,
			Username:                  username,
			Password:                  password,
			AllowedTables:             clickhouse.DefaultAllowedTables(),
			QuotaDurationSeconds:      quotaSettings.quotaDurationSeconds,
			MaxQueriesPerWindow:       quotaSettings.maxQueriesPerWindow,
			MaxExecutionTimePerWindow: quotaSettings.maxExecutionTimePerWindow,
			MaxQueryExecutionTime:     quotaSettings.maxQueryExecutionTime,
			MaxQueryMemoryBytes:       quotaSettings.maxQueryMemoryBytes,
			MaxQueryResultRows:        quotaSettings.maxQueryResultRows,
			RetentionDays:             retentionDays,
		})
	}, restate.WithName("configure clickhouse user"))
	if err != nil {
		return &hydrav1.ConfigureUserResponse{
			Status:   "failed",
			Username: username,
			Error:    fmt.Sprintf("failed to configure clickhouse user: %v", err),
		}, nil
	}

	s.logger.Info("clickhouse user configuration completed successfully",
		"workspace_id", workspaceID,
		"username", username,
		"retention_days", retentionDays,
	)

	return &hydrav1.ConfigureUserResponse{
		Status:   "success",
		Username: username,
	}, nil
}

// decryptPassword is a helper that decrypts a password using the vault service.
// This is used outside of restate.Run when we need the password but don't want to journal it.
func (s *Service) decryptPassword(ctx context.Context, workspaceID, encryptedPassword string) (string, error) {
	resp, err := s.vault.Decrypt(ctx, connect.NewRequest(&vaultv1.DecryptRequest{
		Keyring:   workspaceID,
		Encrypted: encryptedPassword,
	}))
	if err != nil {
		return "", err
	}
	return resp.Msg.GetPlaintext(), nil
}

// quotaSettings holds resolved quota configuration with defaults applied.
type quotaSettings struct {
	quotaDurationSeconds      int32
	maxQueriesPerWindow       int32
	maxExecutionTimePerWindow int32
	maxQueryExecutionTime     int32
	maxQueryMemoryBytes       int64
	maxQueryResultRows        int32
}

// resolveQuotaSettings extracts quota settings from the request, applying defaults where not specified.
func resolveQuotaSettings(req *hydrav1.ConfigureUserRequest) quotaSettings {
	settings := quotaSettings{
		quotaDurationSeconds:      defaultQuotaDurationSeconds,
		maxQueriesPerWindow:       defaultMaxQueriesPerWindow,
		maxExecutionTimePerWindow: defaultMaxExecutionTimePerWindow,
		maxQueryExecutionTime:     defaultMaxQueryExecutionTime,
		maxQueryMemoryBytes:       defaultMaxQueryMemoryBytes,
		maxQueryResultRows:        defaultMaxQueryResultRows,
	}

	if req.QuotaDurationSeconds != nil {
		settings.quotaDurationSeconds = *req.QuotaDurationSeconds
	}
	if req.MaxQueriesPerWindow != nil {
		settings.maxQueriesPerWindow = *req.MaxQueriesPerWindow
	}
	if req.MaxExecutionTimePerWindow != nil {
		settings.maxExecutionTimePerWindow = *req.MaxExecutionTimePerWindow
	}
	if req.MaxQueryExecutionTime != nil {
		settings.maxQueryExecutionTime = *req.MaxQueryExecutionTime
	}
	if req.MaxQueryMemoryBytes != nil {
		settings.maxQueryMemoryBytes = *req.MaxQueryMemoryBytes
	}
	if req.MaxQueryResultRows != nil {
		settings.maxQueryResultRows = *req.MaxQueryResultRows
	}

	return settings
}

// generateSecurePassword creates a cryptographically secure random password.
func generateSecurePassword(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes)[:length], nil
}
