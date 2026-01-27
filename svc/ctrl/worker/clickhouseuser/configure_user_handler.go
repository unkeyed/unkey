package clickhouseuser

import (
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
	"github.com/unkeyed/unkey/pkg/ptr"
)

const (
	defaultQuotaDurationSeconds      = 3600       // 1 hour
	defaultMaxQueriesPerWindow       = 1000       // queries
	defaultMaxExecutionTimePerWindow = 1800       // 30 minutes
	defaultMaxQueryExecutionTime     = 30         // seconds
	defaultMaxQueryMemoryBytes       = 1000000000 // 1 GB
	defaultMaxQueryResultRows        = 10000000   // 10 million rows
	passwordLength                   = 64
)

// existingUserResult wraps the DB lookup to avoid Restate error serialization issues.
type existingUserResult struct {
	Row   db.FindClickhouseWorkspaceSettingsByWorkspaceIDRow
	Found bool
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

// ConfigureUser creates or updates a ClickHouse user for a workspace.
func (s *Service) ConfigureUser(
	ctx restate.ObjectContext,
	req *hydrav1.ConfigureUserRequest,
) (*hydrav1.ConfigureUserResponse, error) {
	workspaceID := restate.Key(ctx)
	s.logger.Info("configuring clickhouse user", "workspace_id", workspaceID)

	quotas := resolveQuotaSettings(req)

	// Check if user exists
	result, err := restate.Run(ctx, func(rc restate.RunContext) (existingUserResult, error) {
		row, err := db.Query.FindClickhouseWorkspaceSettingsByWorkspaceID(rc, s.db.RO(), workspaceID)
		if db.IsNotFound(err) {
			//nolint:exhaustruct // zero value is intentional for not-found case
			return existingUserResult{Found: false}, nil
		}

		if err != nil {
			//nolint:exhaustruct // zero value is intentional for error case
			return existingUserResult{Found: false}, err
		}

		return existingUserResult{Row: row, Found: true}, nil
	}, restate.WithName("check existing"))
	if err != nil {
		return nil, fmt.Errorf("check existing user: %w", err)
	}

	var encryptedPassword string
	var retentionDays int32

	if !result.Found {
		s.logger.Info("creating new user", "workspace_id", workspaceID)

		// Fetch retention days from workspace quota
		quota, err := restate.Run(ctx, func(rc restate.RunContext) (db.Quotum, error) {
			return db.Query.FindQuotaByWorkspaceID(rc, s.db.RO(), workspaceID)
		}, restate.WithName("fetch quota"))
		if err != nil {
			return nil, fmt.Errorf("fetch quota: %w", err)
		}
		retentionDays = quota.LogsRetentionDays

		// Generate, encrypt, and store credentials in one step to avoid journaling plaintext
		encryptedPassword, err = restate.Run(ctx, func(rc restate.RunContext) (string, error) {
			password, err := generateSecurePassword(passwordLength)
			if err != nil {
				return "", fmt.Errorf("generate password: %w", err)
			}

			resp, err := s.vault.Encrypt(rc, connect.NewRequest(&vaultv1.EncryptRequest{
				Keyring: workspaceID,
				Data:    password,
			}))
			if err != nil {
				return "", fmt.Errorf("encrypt password: %w", err)
			}
			encrypted := resp.Msg.GetEncrypted()

			now := time.Now().UnixMilli()
			err = db.Query.InsertClickhouseWorkspaceSettings(rc, s.db.RW(), db.InsertClickhouseWorkspaceSettingsParams{
				WorkspaceID:               workspaceID,
				Username:                  workspaceID,
				PasswordEncrypted:         encrypted,
				QuotaDurationSeconds:      quotas.quotaDurationSeconds,
				MaxQueriesPerWindow:       quotas.maxQueriesPerWindow,
				MaxExecutionTimePerWindow: quotas.maxExecutionTimePerWindow,
				MaxQueryExecutionTime:     quotas.maxQueryExecutionTime,
				MaxQueryMemoryBytes:       quotas.maxQueryMemoryBytes,
				MaxQueryResultRows:        quotas.maxQueryResultRows,
				CreatedAt:                 now,
				UpdatedAt:                 sql.NullInt64{Valid: true, Int64: now},
			})
			if err != nil {
				// Handle crash-recovery: if we inserted but didn't journal, fetch existing
				if db.IsDuplicateKeyError(err) {
					existing, err := db.Query.FindClickhouseWorkspaceSettingsByWorkspaceID(rc, s.db.RO(), workspaceID)
					if err != nil {
						return "", fmt.Errorf("fetch after duplicate: %w", err)
					}

					return existing.ClickhouseWorkspaceSetting.PasswordEncrypted, nil
				}

				return "", fmt.Errorf("insert settings: %w", err)
			}

			return encrypted, nil
		}, restate.WithName("store credentials"))
		if err != nil {
			return nil, err
		}

	} else {
		s.logger.Info("updating existing user", "workspace_id", workspaceID)
		retentionDays = result.Row.Quotas.LogsRetentionDays
		encryptedPassword = result.Row.ClickhouseWorkspaceSetting.PasswordEncrypted

		now := time.Now().UnixMilli()
		_, err = restate.Run(ctx, func(rc restate.RunContext) (restate.Void, error) {
			return restate.Void{}, db.Query.UpdateClickhouseWorkspaceSettingsLimits(rc, s.db.RW(), db.UpdateClickhouseWorkspaceSettingsLimitsParams{
				WorkspaceID:               workspaceID,
				QuotaDurationSeconds:      quotas.quotaDurationSeconds,
				MaxQueriesPerWindow:       quotas.maxQueriesPerWindow,
				MaxExecutionTimePerWindow: quotas.maxExecutionTimePerWindow,
				MaxQueryExecutionTime:     quotas.maxQueryExecutionTime,
				MaxQueryMemoryBytes:       quotas.maxQueryMemoryBytes,
				MaxQueryResultRows:        quotas.maxQueryResultRows,
				UpdatedAt:                 sql.NullInt64{Valid: true, Int64: now},
			})
		}, restate.WithName("update limits"))
		if err != nil {
			return nil, fmt.Errorf("update limits: %w", err)
		}
	}

	// Configure ClickHouse - decrypt inside step to avoid journaling plaintext
	_, err = restate.Run(ctx, func(rc restate.RunContext) (restate.Void, error) {
		resp, err := s.vault.Decrypt(rc, connect.NewRequest(&vaultv1.DecryptRequest{
			Keyring:   workspaceID,
			Encrypted: encryptedPassword,
		}))
		if err != nil {
			return restate.Void{}, fmt.Errorf("decrypt password: %w", err)
		}

		return restate.Void{}, s.clickhouse.ConfigureUser(rc, clickhouse.UserConfig{
			WorkspaceID:               workspaceID,
			Username:                  workspaceID,
			Password:                  resp.Msg.GetPlaintext(),
			AllowedTables:             clickhouse.DefaultAllowedTables(),
			QuotaDurationSeconds:      quotas.quotaDurationSeconds,
			MaxQueriesPerWindow:       quotas.maxQueriesPerWindow,
			MaxExecutionTimePerWindow: quotas.maxExecutionTimePerWindow,
			MaxQueryExecutionTime:     quotas.maxQueryExecutionTime,
			MaxQueryMemoryBytes:       quotas.maxQueryMemoryBytes,
			MaxQueryResultRows:        quotas.maxQueryResultRows,
			RetentionDays:             retentionDays,
		})
	}, restate.WithName("configure clickhouse"))
	if err != nil {
		return nil, fmt.Errorf("configure clickhouse: %w", err)
	}

	s.logger.Info("configured clickhouse user", "workspace_id", workspaceID, "retention_days", retentionDays)

	return &hydrav1.ConfigureUserResponse{}, nil
}

// resolveQuotaSettings applies defaults to unset quota fields.
// Note: We intentionally reject 0 values (which ClickHouse treats as "unlimited")
// to prevent unbounded resource consumption.
func resolveQuotaSettings(req *hydrav1.ConfigureUserRequest) quotaSettings {
	return quotaSettings{
		quotaDurationSeconds:      ptr.PositiveOr(req.QuotaDurationSeconds, defaultQuotaDurationSeconds),
		maxQueriesPerWindow:       ptr.PositiveOr(req.MaxQueriesPerWindow, defaultMaxQueriesPerWindow),
		maxExecutionTimePerWindow: ptr.PositiveOr(req.MaxExecutionTimePerWindow, defaultMaxExecutionTimePerWindow),
		maxQueryExecutionTime:     ptr.PositiveOr(req.MaxQueryExecutionTime, defaultMaxQueryExecutionTime),
		maxQueryMemoryBytes:       ptr.PositiveOr(req.MaxQueryMemoryBytes, defaultMaxQueryMemoryBytes),
		maxQueryResultRows:        ptr.PositiveOr(req.MaxQueryResultRows, defaultMaxQueryResultRows),
	}
}

func generateSecurePassword(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(b)[:length], nil
}
