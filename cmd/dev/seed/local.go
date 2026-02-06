package seed

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/db"
	dbtype "github.com/unkeyed/unkey/pkg/db/types"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/uid"
)

var localCmd = &cli.Command{
	Name:  "local",
	Usage: "Seed database with workspace, project, environment, API, and root key for local development",
	Flags: []cli.Flag{
		cli.String("database-primary", "MySQL database DSN", cli.Default("unkey:password@tcp(127.0.0.1:3306)/unkey?parseTime=true&interpolateParams=true"), cli.EnvVar("UNKEY_DATABASE_PRIMARY")),
		cli.String("slug", "Slug used to generate all IDs and names (e.g., 'flo' creates ws_flo, proj_flo, etc.)", cli.Default("local")),
		cli.String("org-id", "Organization ID for auth matching (defaults to org_localdefault for local auth)", cli.Default("org_localdefault")),
		cli.String("ctrl-url", "Control plane API URL", cli.Default("http://localhost:7091"), cli.EnvVar("UNKEY_CTRL_URL")),
		cli.String("api-key", "API key for control plane authentication", cli.Default("your-local-dev-key"), cli.EnvVar("UNKEY_API_KEY")),
		cli.String("output", "Path to write generated environment variables", cli.Default("dev/.env.seed")),
	},
	Action: seedLocal,
}

func seedLocal(ctx context.Context, cmd *cli.Command) error {
	database, err := db.New(db.Config{
		PrimaryDSN:  cmd.RequireString("database-primary"),
		ReadOnlyDSN: "",
	})
	if err != nil {
		return fmt.Errorf("failed to connect to MySQL: %w", err)
	}

	keyService, err := keys.New(keys.Config{
		DB:           database,
		RateLimiter:  nil,
		RBAC:         nil,
		Clickhouse:   nil,
		Region:       "local",
		UsageLimiter: nil,
		KeyCache:     nil,
	})
	if err != nil {
		return fmt.Errorf("failed to create key service: %w", err)
	}

	slug := cmd.String("slug")
	orgID := cmd.String("org-id")
	now := time.Now().UnixMilli()

	titleCase := strings.ToUpper(slug[:1]) + slug[1:]
	workspaceID := fmt.Sprintf("ws_%s", slug)
	workspaceName := fmt.Sprintf("Org %s", titleCase)

	projectID := uid.New(uid.ProjectPrefix)
	projectSlug := fmt.Sprintf("%s-api", slug)
	projectName := fmt.Sprintf("%s API", titleCase)
	envID := fmt.Sprintf("env_%s", slug)
	rootWorkspaceID := "ws_unkey"
	rootKeySpaceID := fmt.Sprintf("ks_%s_root_keys", slug)
	rootApiID := "api_unkey"
	userKeySpaceID := fmt.Sprintf("ks_%s", slug)
	userApiID := fmt.Sprintf("api_%s", slug)

	rootKeyID := uid.New(uid.KeyPrefix)
	keyResult, err := keyService.CreateKey(ctx, keys.CreateKeyRequest{
		Prefix:     "unkey",
		ByteLength: 16,
	})
	if err != nil {
		return fmt.Errorf("failed to generate root key: %w", err)
	}

	// Create project via control plane API
	logger.Info("creating project via control plane API",
		"workspace", workspaceID,
		"name", projectName,
		"slug", projectSlug,
	)

	err = db.TxRetry(ctx, database.RW(), func(ctx context.Context, tx db.DBTX) error {
		err = db.BulkQuery.UpsertWorkspace(ctx, tx, []db.UpsertWorkspaceParams{
			{
				ID:           workspaceID,
				OrgID:        orgID,
				Name:         workspaceName,
				Slug:         slug,
				CreatedAtM:   now,
				Tier:         sql.NullString{String: "Free", Valid: true},
				BetaFeatures: json.RawMessage(`{"deployments":true}`),
			},
			{
				ID:           rootWorkspaceID,
				OrgID:        fmt.Sprintf("user_%s", slug),
				Name:         "Unkey",
				Slug:         fmt.Sprintf("unkey-%s", slug),
				CreatedAtM:   now,
				Tier:         sql.NullString{String: "Free", Valid: true},
				BetaFeatures: json.RawMessage(`{}`),
			},
		})
		if err != nil {
			return fmt.Errorf("failed to create workspaces: %w", err)
		}

		err = db.Query.InsertProject(ctx, tx, db.InsertProjectParams{
			ID:               projectID,
			WorkspaceID:      workspaceID,
			Name:             projectName,
			Slug:             projectSlug,
			GitRepositoryUrl: sql.NullString{Valid: false, String: ""},
			DefaultBranch:    sql.NullString{Valid: false, String: ""},
			DeleteProtection: sql.NullBool{Valid: false, Bool: false},
			CreatedAt:        time.Now().UnixMilli(),
			UpdatedAt:        sql.NullInt64{Valid: false, Int64: 0},
		})
		if err != nil {
			return fmt.Errorf("failed to create project: %w", err)
		}

		err = db.BulkQuery.InsertEnvironments(ctx, tx, []db.InsertEnvironmentParams{
			{
				ID:             uid.New(uid.EnvironmentPrefix),
				WorkspaceID:    workspaceID,
				ProjectID:      projectID,
				Slug:           "preview",
				Description:    "",
				CreatedAt:      time.Now().UnixMilli(),
				UpdatedAt:      sql.NullInt64{Valid: false, Int64: 0},
				SentinelConfig: []byte{},
			}, {
				ID:             uid.New(uid.EnvironmentPrefix),
				WorkspaceID:    workspaceID,
				ProjectID:      projectID,
				Slug:           "production",
				Description:    "",
				CreatedAt:      time.Now().UnixMilli(),
				UpdatedAt:      sql.NullInt64{Valid: false, Int64: 0},
				SentinelConfig: []byte{},
			},
		})

		err = db.BulkQuery.UpsertQuota(ctx, tx, []db.UpsertQuotaParams{
			{
				WorkspaceID:            workspaceID,
				RequestsPerMonth:       150000,
				AuditLogsRetentionDays: 30,
				LogsRetentionDays:      7,
				Team:                   false,
			},
			{
				WorkspaceID:            rootWorkspaceID,
				RequestsPerMonth:       150000,
				AuditLogsRetentionDays: 30,
				LogsRetentionDays:      7,
				Team:                   false,
			},
		})
		if err != nil {
			return fmt.Errorf("failed to create quotas: %w", err)
		}

		err = db.BulkQuery.UpsertKeySpace(ctx, tx, []db.UpsertKeySpaceParams{
			{
				ID:                 rootKeySpaceID,
				WorkspaceID:        rootWorkspaceID,
				CreatedAtM:         now,
				DefaultPrefix:      sql.NullString{String: "unkey", Valid: true},
				DefaultBytes:       sql.NullInt32{Int32: 16, Valid: true},
				StoreEncryptedKeys: false,
			},
			{
				ID:                 userKeySpaceID,
				WorkspaceID:        workspaceID,
				CreatedAtM:         now,
				DefaultPrefix:      sql.NullString{String: "sk", Valid: true},
				DefaultBytes:       sql.NullInt32{Int32: 16, Valid: true},
				StoreEncryptedKeys: true,
			},
		})
		if err != nil {
			return fmt.Errorf("failed to create key spaces: %w", err)
		}

		err = db.BulkQuery.InsertApis(ctx, tx, []db.InsertApiParams{
			{
				ID:          rootApiID,
				Name:        "Unkey",
				WorkspaceID: rootWorkspaceID,
				AuthType:    db.NullApisAuthType{Valid: true, ApisAuthType: db.ApisAuthTypeKey},
				IpWhitelist: sql.NullString{},
				KeyAuthID:   sql.NullString{String: rootKeySpaceID, Valid: true},
				CreatedAtM:  now,
			},
			{
				ID:          userApiID,
				Name:        fmt.Sprintf("%s API", titleCase),
				WorkspaceID: workspaceID,
				AuthType:    db.NullApisAuthType{Valid: true, ApisAuthType: db.ApisAuthTypeKey},
				IpWhitelist: sql.NullString{},
				KeyAuthID:   sql.NullString{String: userKeySpaceID, Valid: true},
				CreatedAtM:  now,
			},
		})
		if err != nil && !db.IsDuplicateKeyError(err) {
			return fmt.Errorf("failed to create APIs: %w", err)
		}

		err = db.Query.InsertKey(ctx, tx, db.InsertKeyParams{
			ID:                 rootKeyID,
			KeySpaceID:         rootKeySpaceID,
			Hash:               keyResult.Hash,
			Start:              keyResult.Start,
			WorkspaceID:        rootWorkspaceID,
			ForWorkspaceID:     sql.NullString{String: workspaceID, Valid: true},
			Name:               sql.NullString{String: fmt.Sprintf("%s Dev Root Key", titleCase), Valid: true},
			IdentityID:         sql.NullString{},
			Meta:               sql.NullString{},
			Expires:            sql.NullTime{},
			CreatedAtM:         now,
			Enabled:            true,
			RemainingRequests:  sql.NullInt32{},
			RefillDay:          sql.NullInt16{},
			RefillAmount:       sql.NullInt32{},
			PendingMigrationID: sql.NullString{},
		})
		if err != nil && !db.IsDuplicateKeyError(err) {
			return fmt.Errorf("failed to create root key: %w", err)
		}

		allPermissions := []string{
			"api.*.create_api",
			"api.*.read_api",
			"api.*.delete_api",
			"api.*.create_key",
			"api.*.read_key",
			"api.*.update_key",
			"api.*.delete_key",
			"api.*.verify_key",
			"api.*.decrypt_key",
			"api.*.encrypt_key",
			"api.*.read_analytics",
			"identity.*.create_identity",
			"identity.*.read_identity",
			"identity.*.update_identity",
			"identity.*.delete_identity",
			"rbac.*.create_permission",
			"rbac.*.read_permission",
			"rbac.*.delete_permission",
			"rbac.*.create_role",
			"rbac.*.read_role",
			"rbac.*.delete_role",
			"rbac.*.add_permission_to_key",
			"rbac.*.remove_permission_from_key",
			"rbac.*.add_role_to_key",
			"rbac.*.remove_role_from_key",
			"ratelimit.*.create_namespace",
			"ratelimit.*.limit",
			"ratelimit.*.read_override",
			"ratelimit.*.set_override",
			"workspace.*.read_workspace",
			"project.*.generate_upload_url",
			"project.*.create_deployment",
			"project.*.read_deployment",
		}

		permissionParams := make([]db.InsertPermissionParams, len(allPermissions))
		permissionIDs := make([]string, len(allPermissions))
		for i, perm := range allPermissions {
			permID := uid.New(uid.PermissionPrefix)
			permissionIDs[i] = permID
			permissionParams[i] = db.InsertPermissionParams{
				PermissionID: permID,
				WorkspaceID:  rootWorkspaceID,
				Name:         perm,
				Slug:         perm,
				Description:  dbtype.NullString{Valid: false, String: ""},
				CreatedAtM:   now,
			}
		}

		err = db.BulkQuery.InsertPermissions(ctx, tx, permissionParams)
		if err != nil && !db.IsDuplicateKeyError(err) {
			return fmt.Errorf("failed to insert permissions: %w", err)
		}

		keyPermissionParams := make([]db.InsertKeyPermissionParams, len(allPermissions))
		for i := range allPermissions {
			keyPermissionParams[i] = db.InsertKeyPermissionParams{
				KeyID:        rootKeyID,
				PermissionID: permissionIDs[i],
				WorkspaceID:  rootWorkspaceID,
				CreatedAt:    now,
				UpdatedAt:    sql.NullInt64{},
			}
		}

		err = db.BulkQuery.InsertKeyPermissions(ctx, tx, keyPermissionParams)
		if err != nil && !db.IsDuplicateKeyError(err) {
			return fmt.Errorf("failed to insert key permissions: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	logger.Info("project created successfully via control plane", "id", projectID)

	// Write environment file with generated values
	if outputFile := cmd.String("output"); outputFile != "" {
		envContent := fmt.Sprintf(`# Generated by: unkey dev seed local --slug=%s
# Source this file or copy values to your .env

UNKEY_WORKSPACE_ID=%s
UNKEY_PROJECT_ID=%s
UNKEY_API_ID=%s
UNKEY_KEYSPACE_ID=%s
UNKEY_ROOT_KEY=%s
`,
			slug,
			workspaceID,
			projectID,
			userApiID,
			userKeySpaceID,
			keyResult.Key,
		)

		// Ensure directory exists
		if dir := filepath.Dir(outputFile); dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}
		}

		if err := os.WriteFile(outputFile, []byte(envContent), 0600); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		logger.Info("wrote environment file", "path", outputFile)
	}

	logger.Info("seed completed",
		"workspace", workspaceID,
		"project", projectID,
		"environment", envID,
		"api", userApiID,
		"keySpace", userKeySpaceID,
		"rootKey", keyResult.Key,
	)

	return nil
}
