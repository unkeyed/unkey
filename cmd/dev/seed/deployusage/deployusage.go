package deployusage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	driver "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/db"
	dbtype "github.com/unkeyed/unkey/pkg/db/types"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/uid"
)

const (
	resourcePrefix      = "seed-deploy-usage/"
	deploymentK8sPrefix = "seed-deploy-usage-"
	productionSlug      = "production"
	previewSlug         = "preview"
)

var Cmd = &cli.Command{
	Name:        "deploy-usage",
	Usage:       "Seed realistic multi-app data for the Deploy usage dashboard",
	Description: "Creates 15 apps with production and preview environments, deterministic deployment history, and three months of hourly usage.",
	Flags: []cli.Flag{
		cli.String("workspace", "Workspace ID to seed", cli.Default("ws_local")),
		cli.String("project", "Project ID or slug to seed", cli.Default("local-api")),
		cli.String("clickhouse-url", "ClickHouse URL", cli.Default("clickhouse://default:password@127.0.0.1:9000")),
		cli.String("database-primary", "MySQL database DSN", cli.Default("unkey:password@tcp(127.0.0.1:3306)/unkey?parseTime=true&interpolateParams=true"), cli.EnvVar("UNKEY_DATABASE_PRIMARY")),
	},
	Action: seed,
}

type target struct {
	profile      appProfile
	appID        string
	productionID string
	previewID    string
	workspaceID  string
	projectID    string
}

func seed(ctx context.Context, cmd *cli.Command) error {
	database, err := db.New(db.Config{
		PrimaryDSN:  cmd.RequireString("database-primary"),
		ReadOnlyDSN: "",
		Tags:        sqlcomment.Disabled(),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to MySQL: %w", err)
	}
	defer func() { _ = database.Close() }()

	workspaceID := cmd.RequireString("workspace")
	project, err := db.Query.FindProjectByIdOrSlug(ctx, database.RO(), db.FindProjectByIdOrSlugParams{
		WorkspaceID: workspaceID,
		Project:     cmd.RequireString("project"),
	})
	if err != nil {
		return fmt.Errorf("failed to find project: %w", err)
	}

	now := time.Now().UTC()
	targets, err := db.TxWithResult(ctx, database.RW(), func(ctx context.Context, tx db.DBTX) ([]target, error) {
		targets, err := ensureTargets(ctx, tx, workspaceID, project.ID, now.UnixMilli())
		if err != nil {
			return nil, err
		}
		if err := replaceDeployments(ctx, tx, workspaceID, generateDeployments(now, targets)); err != nil {
			return nil, err
		}
		return targets, nil
	})
	if err != nil {
		return fmt.Errorf("failed to create demo resources: %w", err)
	}

	ch, err := clickhouse.New(clickhouse.Config{URL: cmd.RequireString("clickhouse-url")})
	if err != nil {
		return fmt.Errorf("failed to connect to ClickHouse: %w", err)
	}
	defer func() { _ = ch.Close() }()

	if err := replaceUsage(ctx, ch, workspaceID, generateUsage(now, targets)); err != nil {
		return err
	}

	logger.Info("seeded Deploy usage dashboard data",
		"workspace_id", workspaceID,
		"project_id", project.ID,
		"apps", len(targets),
		"environments", len(targets)*2,
		"deployments", len(generateDeployments(now, targets)),
	)
	return nil
}

func replaceDeployments(
	ctx context.Context,
	tx db.DBTX,
	workspaceID string,
	deployments []deploymentSeed,
) error {
	_, err := tx.ExecContext(
		ctx,
		"DELETE FROM deployments WHERE workspace_id = ? AND k8s_name LIKE ?",
		workspaceID,
		deploymentK8sPrefix+"%",
	)
	if err != nil {
		return fmt.Errorf("failed to clear existing demo deployments: %w", err)
	}

	for _, deployment := range deployments {
		err = db.Query.InsertDeployment(ctx, tx, db.InsertDeploymentParams{
			ID:                            deployment.id,
			K8sName:                       deployment.k8sName,
			WorkspaceID:                   deployment.workspaceID,
			ProjectID:                     deployment.projectID,
			AppID:                         deployment.appID,
			EnvironmentID:                 deployment.environmentID,
			GitCommitSha:                  sql.NullString{String: deployment.commitSHA, Valid: true},
			GitBranch:                     sql.NullString{String: "main", Valid: true},
			SentinelConfig:                []byte("{}"),
			GitCommitMessage:              sql.NullString{String: deployment.message, Valid: true},
			GitCommitAuthorHandle:         sql.NullString{String: "deploy-usage-seed", Valid: true},
			GitCommitAuthorAvatarUrl:      sql.NullString{},
			GitCommitTimestamp:            sql.NullInt64{Int64: deployment.createdAt, Valid: true},
			EncryptedEnvironmentVariables: []byte{},
			Command:                       dbtype.StringSlice{},
			Status:                        mysqltype.DeploymentsStatusReady,
			CpuMillicores:                 500,
			MemoryMib:                     512,
			StorageMib:                    1024,
			Port:                          8080,
			ShutdownSignal:                db.DeploymentsShutdownSignalSIGTERM,
			UpstreamProtocol:              db.DeploymentsUpstreamProtocolHttp1,
			Healthcheck:                   dbtype.NullHealthcheck{Healthcheck: nil, Valid: false},
			PrNumber:                      sql.NullInt64{},
			ForkRepositoryFullName:        sql.NullString{},
			DeploymentTrigger:             db.DeploymentsTriggerGithub,
			TriggeredBy:                   sql.NullString{String: "deploy-usage-seed", Valid: true},
			TriggerReason:                 sql.NullString{},
			CreatedAt:                     deployment.createdAt,
			UpdatedAt:                     sql.NullInt64{},
		})
		if err != nil {
			return fmt.Errorf("failed to create demo deployment %s: %w", deployment.id, err)
		}
	}
	return nil
}

func ensureTargets(ctx context.Context, tx db.DBTX, workspaceID, projectID string, now int64) ([]target, error) {
	targets := make([]target, 0, len(appProfiles))
	for _, profile := range appProfiles {
		appID, err := ensureApp(ctx, tx, workspaceID, projectID, profile, now)
		if err != nil {
			return nil, err
		}

		productionID, err := ensureEnvironment(ctx, tx, workspaceID, projectID, appID, productionSlug, mysqltype.EnvironmentKindProduction, now)
		if err != nil {
			return nil, err
		}
		previewID, err := ensureEnvironment(ctx, tx, workspaceID, projectID, appID, previewSlug, mysqltype.EnvironmentKindPreview, now)
		if err != nil {
			return nil, err
		}

		targets = append(targets, target{
			profile:      profile,
			appID:        appID,
			productionID: productionID,
			previewID:    previewID,
			workspaceID:  workspaceID,
			projectID:    projectID,
		})
	}
	return targets, nil
}

func ensureApp(ctx context.Context, tx db.DBTX, workspaceID, projectID string, profile appProfile, now int64) (string, error) {
	app, err := db.Query.FindAppByProjectAndSlug(ctx, tx, db.FindAppByProjectAndSlugParams{
		ProjectID: projectID,
		Slug:      profile.slug,
	})
	if err == nil {
		return app.ID, nil
	}
	if err != sql.ErrNoRows {
		return "", fmt.Errorf("failed to find app %q: %w", profile.name, err)
	}

	appID := uid.New(uid.AppPrefix)
	err = db.Query.InsertApp(ctx, tx, db.InsertAppParams{
		ID:               appID,
		WorkspaceID:      workspaceID,
		ProjectID:        projectID,
		Name:             profile.name,
		Slug:             profile.slug,
		DefaultBranch:    "main",
		DeleteProtection: sql.NullBool{},
		CreatedAt:        now,
		UpdatedAt:        sql.NullInt64{},
	})
	if err != nil {
		return "", fmt.Errorf("failed to create app %q: %w", profile.name, err)
	}
	return appID, nil
}

func ensureEnvironment(
	ctx context.Context,
	tx db.DBTX,
	workspaceID string,
	projectID string,
	appID string,
	slug string,
	kind mysqltype.EnvironmentKind,
	now int64,
) (string, error) {
	environment, err := db.Query.FindEnvironmentByAppIdAndSlug(ctx, tx, db.FindEnvironmentByAppIdAndSlugParams{
		AppID: appID,
		Slug:  slug,
	})
	if err == nil {
		return environment.ID, nil
	}
	if err != sql.ErrNoRows {
		return "", fmt.Errorf("failed to find %s environment for app %s: %w", slug, appID, err)
	}

	environmentID := uid.New(uid.EnvironmentPrefix)
	err = db.Query.InsertEnvironment(ctx, tx, db.InsertEnvironmentParams{
		ID:          environmentID,
		WorkspaceID: workspaceID,
		ProjectID:   projectID,
		AppID:       appID,
		Slug:        slug,
		Description: "",
		Kind:        kind,
		CreatedAt:   now,
		UpdatedAt:   sql.NullInt64{},
	})
	if err != nil {
		return "", fmt.Errorf("failed to create %s environment for app %s: %w", slug, appID, err)
	}
	return environmentID, nil
}

func replaceUsage(ctx context.Context, ch *clickhouse.Client, workspaceID string, rows []usageRow) error {
	err := ch.Exec(ctx, `
		ALTER TABLE default.instance_usage_per_hour_v1
		DELETE WHERE workspace_id = {workspace_id:String}
		  AND startsWith(resource_id, {resource_prefix:String})
		SETTINGS mutations_sync = 1
	`, driver.Named("workspace_id", workspaceID), driver.Named("resource_prefix", resourcePrefix))
	if err != nil {
		return fmt.Errorf("failed to clear existing demo usage: %w", err)
	}

	batch, err := ch.Conn().PrepareBatch(ctx, `
		INSERT INTO default.instance_usage_per_hour_v1 (
			time, workspace_id, project_id, app_id, environment_id,
			resource_type, resource_id, container_uid, instance_id,
			cpu_seconds, memory_gib_hours, disk_gib_hours,
			network_egress_public_bytes, network_egress_private_bytes,
			network_ingress_public_bytes, network_ingress_private_bytes,
			sample_pairs
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare demo usage batch: %w", err)
	}
	defer func() { _ = batch.Close() }()

	for _, row := range rows {
		err = batch.Append(
			row.time,
			row.workspaceID,
			row.projectID,
			row.appID,
			row.environmentID,
			"deployment",
			row.resourceID,
			row.containerUID,
			row.instanceID,
			row.cpuSeconds,
			row.memoryGiBHours,
			row.diskGiBHours,
			row.egressPublicBytes,
			row.egressPrivateBytes,
			row.ingressPublicBytes,
			row.ingressPrivateBytes,
			row.samplePairs,
		)
		if err != nil {
			return fmt.Errorf("failed to append demo usage: %w", err)
		}
	}
	if err := batch.Send(); err != nil {
		return fmt.Errorf("failed to insert demo usage: %w", err)
	}
	return nil
}
