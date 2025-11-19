package seed

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/pkg/cli"
	"github.com/unkeyed/unkey/go/pkg/db"
	dbtype "github.com/unkeyed/unkey/go/pkg/db/types"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

var localCmd = &cli.Command{
	Name:  "local",
	Usage: "Seed database with workspace, project, environment, API, and root key for local development",
	Flags: []cli.Flag{
		cli.String("database-primary", "MySQL database DSN", cli.Default("unkey:password@tcp(127.0.0.1:3306)/unkey?parseTime=true&interpolateParams=true"), cli.EnvVar("UNKEY_DATABASE_PRIMARY")),
		cli.String("slug", "Slug used to generate all IDs and names (e.g., 'flo' creates ws_flo, proj_flo, etc.)", cli.Default("local")),
		cli.String("org-id", "Organization ID for auth matching (defaults to org_localdefault for local auth)", cli.Default("org_localdefault")),
	},
	Action: seedLocal,
}

func seedLocal(ctx context.Context, cmd *cli.Command) error {
	logger := logging.New()

	// Connect to MySQL
	database, err := db.New(db.Config{
		PrimaryDSN:  cmd.RequireString("database-primary"),
		ReadOnlyDSN: "",
		Logger:      logger,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to MySQL: %w", err)
	}

	// Create keys service for proper key generation
	keyService, err := keys.New(keys.Config{
		Logger:       logger,
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

	// Generate all IDs and names from slug
	titleCase := strings.ToUpper(slug[:1]) + slug[1:]
	workspaceID := fmt.Sprintf("ws_%s", slug)
	workspaceName := fmt.Sprintf("Org %s", titleCase)
	projectID := fmt.Sprintf("proj_%s", slug)
	envID := fmt.Sprintf("env_%s", slug)
	rootWorkspaceID := fmt.Sprintf("ws_%s_root", slug)
	rootKeySpaceID := fmt.Sprintf("ks_%s_root_keys", slug)
	rootApiID := fmt.Sprintf("api_%s_root_keys", slug)
	userKeySpaceID := fmt.Sprintf("ks_%s", slug)
	userApiID := fmt.Sprintf("api_%s", slug)

	// Generate root key using keys service
	rootKeyID := uid.New(uid.KeyPrefix)
	keyResult, err := keyService.CreateKey(ctx, keys.CreateKeyRequest{
		Prefix:     "unkey",
		ByteLength: 16,
	})
	if err != nil {
		return fmt.Errorf("failed to generate root key: %w", err)
	}

	err = db.Tx(ctx, database.RW(), func(ctx context.Context, tx db.DBTX) error {
		// 1. Create user workspace with beta features
		logger.Info("creating workspace", "id", workspaceID, "orgId", orgID)
		err := db.Query.UpsertWorkspace(ctx, tx, db.UpsertWorkspaceParams{
			ID:           workspaceID,
			OrgID:        orgID,
			Name:         workspaceName,
			Slug:         slug,
			CreatedAtM:   now,
			Tier:         sql.NullString{String: "Free", Valid: true},
			BetaFeatures: json.RawMessage(`{"deployments":true}`),
		})
		if err != nil {
			return fmt.Errorf("failed to create workspace: %w", err)
		}

		// 1b. Create quota for user workspace
		logger.Info("creating quota for user workspace")
		err = db.Query.UpsertQuota(ctx, tx, db.UpsertQuotaParams{
			WorkspaceID:            workspaceID,
			RequestsPerMonth:       150000,
			AuditLogsRetentionDays: 30,
			LogsRetentionDays:      7,
			Team:                   false,
		})
		if err != nil {
			return fmt.Errorf("failed to create user workspace quota: %w", err)
		}

		// 2. Create project
		logger.Info("creating project", "id", projectID)
		err = db.Query.InsertProject(ctx, tx, db.InsertProjectParams{
			ID:               projectID,
			WorkspaceID:      workspaceID,
			Name:             fmt.Sprintf("%s API", titleCase),
			Slug:             fmt.Sprintf("%s-api", slug),
			GitRepositoryUrl: sql.NullString{},
			DefaultBranch:    sql.NullString{},
			DeleteProtection: sql.NullBool{},
			CreatedAt:        now,
			UpdatedAt:        sql.NullInt64{},
		})
		if err != nil && !db.IsDuplicateKeyError(err) {
			return fmt.Errorf("failed to create project: %w", err)
		}

		// 3. Create environment
		logger.Info("creating environment", "id", envID)
		err = db.Query.UpsertEnvironment(ctx, tx, db.UpsertEnvironmentParams{
			ID:            envID,
			WorkspaceID:   workspaceID,
			ProjectID:     projectID,
			Slug:          "development",
			GatewayConfig: []byte("{}"),
			CreatedAt:     now,
		})
		if err != nil {
			return fmt.Errorf("failed to create environment: %w", err)
		}

		// 4. Create root workspace - matches deployment/04-seed-workspace.sql
		logger.Info("creating root workspace", "id", rootWorkspaceID)
		err = db.Query.UpsertWorkspace(ctx, tx, db.UpsertWorkspaceParams{
			ID:           rootWorkspaceID,
			OrgID:        fmt.Sprintf("user_%s", slug),
			Name:         "Unkey",
			Slug:         fmt.Sprintf("unkey-%s", slug),
			CreatedAtM:   now,
			Tier:         sql.NullString{String: "Free", Valid: true},
			BetaFeatures: json.RawMessage(`{}`),
		})
		if err != nil {
			return fmt.Errorf("failed to create root workspace: %w", err)
		}

		// 4b. Create quota for root workspace
		logger.Info("creating quota for root workspace")
		err = db.Query.UpsertQuota(ctx, tx, db.UpsertQuotaParams{
			WorkspaceID:            rootWorkspaceID,
			RequestsPerMonth:       150000,
			AuditLogsRetentionDays: 30,
			LogsRetentionDays:      7,
			Team:                   false,
		})
		if err != nil {
			return fmt.Errorf("failed to create quota: %w", err)
		}

		// 5. Create root key space and API - matches deployment/04-seed-workspace.sql
		logger.Info("creating root API and key space", "apiId", rootApiID)
		err = db.Query.UpsertKeySpace(ctx, tx, db.UpsertKeySpaceParams{
			ID:                 rootKeySpaceID,
			WorkspaceID:        rootWorkspaceID,
			CreatedAtM:         now,
			DefaultPrefix:      sql.NullString{String: "unkey", Valid: true},
			DefaultBytes:       sql.NullInt32{Int32: 16, Valid: true},
			StoreEncryptedKeys: false,
		})
		if err != nil {
			return fmt.Errorf("failed to create root key space: %w", err)
		}

		err = db.Query.InsertApi(ctx, tx, db.InsertApiParams{
			ID:          rootApiID,
			Name:        "Unkey",
			WorkspaceID: rootWorkspaceID,
			AuthType:    db.NullApisAuthType{Valid: true, ApisAuthType: db.ApisAuthTypeKey},
			IpWhitelist: sql.NullString{},
			KeyAuthID:   sql.NullString{String: rootKeySpaceID, Valid: true},
			CreatedAtM:  now,
		})
		if err != nil && !db.IsDuplicateKeyError(err) {
			return fmt.Errorf("failed to create root API: %w", err)
		}

		// 6. Create user API and key space with encrypted keys enabled
		logger.Info("creating user API and key space", "apiId", userApiID)
		err = db.Query.UpsertKeySpace(ctx, tx, db.UpsertKeySpaceParams{
			ID:                 userKeySpaceID,
			WorkspaceID:        workspaceID,
			CreatedAtM:         now,
			DefaultPrefix:      sql.NullString{String: "sk", Valid: true},
			DefaultBytes:       sql.NullInt32{Int32: 16, Valid: true},
			StoreEncryptedKeys: true,
		})
		if err != nil {
			return fmt.Errorf("failed to create user key space: %w", err)
		}

		err = db.Query.InsertApi(ctx, tx, db.InsertApiParams{
			ID:          userApiID,
			Name:        fmt.Sprintf("%s API", titleCase),
			WorkspaceID: workspaceID,
			AuthType:    db.NullApisAuthType{Valid: true, ApisAuthType: db.ApisAuthTypeKey},
			IpWhitelist: sql.NullString{},
			KeyAuthID:   sql.NullString{String: userKeySpaceID, Valid: true},
			CreatedAtM:  now,
		})
		if err != nil && !db.IsDuplicateKeyError(err) {
			return fmt.Errorf("failed to create user API: %w", err)
		}

		// 7. Create root key with all permissions
		logger.Info("creating root key", "keyId", rootKeyID)
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

		// 8. Add all permissions to root key using bulk insert
		allPermissions := []string{
			// API permissions
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
			// Identity permissions
			"identity.*.create_identity",
			"identity.*.read_identity",
			"identity.*.update_identity",
			"identity.*.delete_identity",
			// RBAC permissions
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
			// Ratelimit permissions
			"ratelimit.*.create_namespace",
			"ratelimit.*.limit",
			"ratelimit.*.read_override",
			"ratelimit.*.set_override",
			// Workspace permissions
			"workspace.*.read_workspace",
		}

		logger.Info("adding permissions to root key", "count", len(allPermissions))

		// Build permission params for bulk insert
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
				Description:  dbtype.NullString{},
				CreatedAtM:   now,
			}
		}

		// Bulk insert all permissions
		err = db.BulkQuery.InsertPermissions(ctx, tx, permissionParams)
		if err != nil && !db.IsDuplicateKeyError(err) {
			return fmt.Errorf("failed to bulk insert permissions: %w", err)
		}

		// Build key permission params for bulk insert
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

		// Bulk insert all key permissions
		err = db.BulkQuery.InsertKeyPermissions(ctx, tx, keyPermissionParams)
		if err != nil && !db.IsDuplicateKeyError(err) {
			return fmt.Errorf("failed to bulk insert key permissions: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	// Print summary
	logger.Info("local development seed completed successfully")
	logger.Info("workspace", "id", workspaceID)
	logger.Info("project", "id", projectID)
	logger.Info("environment", "id", envID)
	logger.Info("api", "id", userApiID)
	logger.Info("keySpace", "id", userKeySpaceID)
	logger.Info("rootKey", "key", keyResult.Key, "keyId", rootKeyID)

	return nil
}
