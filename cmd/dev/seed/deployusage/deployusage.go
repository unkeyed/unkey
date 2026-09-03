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
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/uid"
)

const (
	resourcePrefix = "seed-deploy-usage/"
	productionSlug = "production"
	previewSlug    = "preview"
)

var Cmd = &cli.Command{
	Name:        "deploy-usage",
	Usage:       "Seed realistic multi-app data for the Deploy usage dashboard",
	Description: "Creates 15 apps with production and preview environments, then writes three months of deterministic hourly usage to the dashboard rollup.",
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

	targets, err := db.TxWithResult(ctx, database.RW(), func(ctx context.Context, tx db.DBTX) ([]target, error) {
		return ensureTargets(ctx, tx, workspaceID, project.ID, time.Now().UnixMilli())
	})
	if err != nil {
		return fmt.Errorf("failed to create demo apps: %w", err)
	}

	ch, err := clickhouse.New(clickhouse.Config{URL: cmd.RequireString("clickhouse-url")})
	if err != nil {
		return fmt.Errorf("failed to connect to ClickHouse: %w", err)
	}
	defer func() { _ = ch.Close() }()

	if err := replaceUsage(ctx, ch, workspaceID, generateUsage(time.Now().UTC(), targets)); err != nil {
		return err
	}

	logger.Info("seeded Deploy usage dashboard data",
		"workspace_id", workspaceID,
		"project_id", project.ID,
		"apps", len(targets),
		"environments", len(targets)*2,
	)
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
