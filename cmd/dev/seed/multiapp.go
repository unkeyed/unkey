package seed

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/db"
	dbtype "github.com/unkeyed/unkey/pkg/db/types"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/uid"
)

var multiAppCmd = &cli.Command{
	Name:  "multi-app",
	Usage: "Seed a project with two apps (api + worker) to test multi-app deployments and network isolation",
	Flags: []cli.Flag{
		cli.String("database-primary", "MySQL database DSN", cli.Default("unkey:password@tcp(127.0.0.1:3306)/unkey?parseTime=true&interpolateParams=true"), cli.EnvVar("UNKEY_DATABASE_PRIMARY")),
		cli.String("slug", "Slug used for workspace/project naming", cli.Default("local")),
	},
	Action: seedMultiApp,
}

func seedMultiApp(ctx context.Context, cmd *cli.Command) error {
	database, err := db.New(db.Config{
		PrimaryDSN:  cmd.RequireString("database-primary"),
		ReadOnlyDSN: "",
	})
	if err != nil {
		return fmt.Errorf("failed to connect to MySQL: %w", err)
	}

	slug := cmd.String("slug")
	now := time.Now().UnixMilli()

	titleCase := strings.ToUpper(slug[:1]) + slug[1:]
	workspaceID := fmt.Sprintf("ws_%s", slug)

	projectID := uid.New(uid.ProjectPrefix)
	projectSlug := fmt.Sprintf("%s-multi", slug)
	projectName := fmt.Sprintf("%s Multi-App", titleCase)

	apiAppID := uid.New(uid.AppPrefix)
	workerAppID := uid.New(uid.AppPrefix)

	previewEnvID := uid.New(uid.EnvironmentPrefix)
	productionEnvID := uid.New(uid.EnvironmentPrefix)

	err = db.TxRetry(ctx, database.RW(), func(ctx context.Context, tx db.DBTX) error {
		// Create project
		err = db.Query.InsertProject(ctx, tx, db.InsertProjectParams{
			ID:               projectID,
			WorkspaceID:      workspaceID,
			Name:             projectName,
			Slug:             projectSlug,
			DefaultBranch:    sql.NullString{Valid: false, String: ""},
			DeleteProtection: sql.NullBool{Valid: false, Bool: false},
			CreatedAt:        now,
			UpdatedAt:        sql.NullInt64{Valid: false, Int64: 0},
		})
		if err != nil {
			return fmt.Errorf("failed to create project: %w", err)
		}

		// Create two apps: api and worker
		err = db.BulkQuery.InsertApps(ctx, tx, []db.InsertAppParams{
			{
				ID:               apiAppID,
				WorkspaceID:      workspaceID,
				ProjectID:        projectID,
				Name:             "API",
				Slug:             "api",
				LiveDeploymentID: sql.NullString{},
				IsRolledBack:     false,
				DepotProjectID:   sql.NullString{},
				DeleteProtection: sql.NullBool{Valid: false, Bool: false},
				CreatedAt:        now,
				UpdatedAt:        sql.NullInt64{Valid: false, Int64: 0},
			},
			{
				ID:               workerAppID,
				WorkspaceID:      workspaceID,
				ProjectID:        projectID,
				Name:             "Worker",
				Slug:             "worker",
				LiveDeploymentID: sql.NullString{},
				IsRolledBack:     false,
				DepotProjectID:   sql.NullString{},
				DeleteProtection: sql.NullBool{Valid: false, Bool: false},
				CreatedAt:        now,
				UpdatedAt:        sql.NullInt64{Valid: false, Int64: 0},
			},
		})
		if err != nil {
			return fmt.Errorf("failed to create apps: %w", err)
		}

		// Create environments
		err = db.BulkQuery.InsertEnvironments(ctx, tx, []db.InsertEnvironmentParams{
			{
				ID:          previewEnvID,
				WorkspaceID: workspaceID,
				ProjectID:   projectID,
				Slug:        "preview",
				Description: "",
				CreatedAt:   now,
				UpdatedAt:   sql.NullInt64{Valid: false, Int64: 0},
			},
			{
				ID:          productionEnvID,
				WorkspaceID: workspaceID,
				ProjectID:   projectID,
				Slug:        "production",
				Description: "",
				CreatedAt:   now,
				UpdatedAt:   sql.NullInt64{Valid: false, Int64: 0},
			},
		})
		if err != nil {
			return fmt.Errorf("failed to create environments: %w", err)
		}

		// Build settings: same Dockerfile for both apps, command override picks the service
		err = db.BulkQuery.UpsertAppBuildSettings(ctx, tx, []db.UpsertAppBuildSettingsParams{
			// API app — preview + production
			{
				WorkspaceID:   workspaceID,
				AppID:         apiAppID,
				EnvironmentID: previewEnvID,
				Dockerfile:    "svc/api/Dockerfile",
				DockerContext: ".",
				CreatedAt:     now,
				UpdatedAt:     sql.NullInt64{Valid: true, Int64: now},
			},
			{
				WorkspaceID:   workspaceID,
				AppID:         apiAppID,
				EnvironmentID: productionEnvID,
				Dockerfile:    "svc/api/Dockerfile",
				DockerContext: ".",
				CreatedAt:     now,
				UpdatedAt:     sql.NullInt64{Valid: true, Int64: now},
			},
			// Worker app — same Dockerfile, override command in runtime settings
			{
				WorkspaceID:   workspaceID,
				AppID:         workerAppID,
				EnvironmentID: previewEnvID,
				Dockerfile:    "svc/api/Dockerfile",
				DockerContext: ".",
				CreatedAt:     now,
				UpdatedAt:     sql.NullInt64{Valid: true, Int64: now},
			},
			{
				WorkspaceID:   workspaceID,
				AppID:         workerAppID,
				EnvironmentID: productionEnvID,
				Dockerfile:    "svc/api/Dockerfile",
				DockerContext: ".",
				CreatedAt:     now,
				UpdatedAt:     sql.NullInt64{Valid: true, Int64: now},
			},
		})
		if err != nil {
			return fmt.Errorf("failed to create build settings: %w", err)
		}

		// GitHub repo connections: one per app, pointing at a test mono-repo
		err = db.Query.InsertGithubRepoConnection(ctx, tx, db.InsertGithubRepoConnectionParams{
			ProjectID:          projectID,
			AppID:              apiAppID,
			InstallationID:     1,
			RepositoryID:       1,
			RepositoryFullName: "Flo4604/mono-repo-test",
			CreatedAt:          now,
			UpdatedAt:          sql.NullInt64{Valid: false, Int64: 0},
		})
		if err != nil {
			return fmt.Errorf("failed to create github repo connection for api app: %w", err)
		}

		err = db.Query.InsertGithubRepoConnection(ctx, tx, db.InsertGithubRepoConnectionParams{
			ProjectID:          projectID,
			AppID:              workerAppID,
			InstallationID:     1,
			RepositoryID:       1,
			RepositoryFullName: "Flo4604/mono-repo-test",
			CreatedAt:          now,
			UpdatedAt:          sql.NullInt64{Valid: false, Int64: 0},
		})
		if err != nil {
			return fmt.Errorf("failed to create github repo connection for worker app: %w", err)
		}

		// Runtime settings: both apps on port 8080, 256 CPU/mem
		err = db.BulkQuery.UpsertAppRuntimeSettings(ctx, tx, []db.UpsertAppRuntimeSettingsParams{
			// API app — preview + production
			{
				WorkspaceID:    workspaceID,
				AppID:          apiAppID,
				EnvironmentID:  previewEnvID,
				Port:           8080,
				CpuMillicores:  256,
				MemoryMib:      256,
				Command:        dbtype.StringSlice{},
				Healthcheck:    dbtype.NullHealthcheck{Healthcheck: nil, Valid: false},
				RegionConfig:   dbtype.RegionConfig{},
				SentinelConfig: []byte{},
				ShutdownSignal: db.AppRuntimeSettingsShutdownSignalSIGTERM,
				CreatedAt:      now,
				UpdatedAt:      sql.NullInt64{Valid: true, Int64: now},
			},
			{
				WorkspaceID:    workspaceID,
				AppID:          apiAppID,
				EnvironmentID:  productionEnvID,
				Port:           8080,
				CpuMillicores:  256,
				MemoryMib:      256,
				Command:        dbtype.StringSlice{},
				Healthcheck:    dbtype.NullHealthcheck{Healthcheck: nil, Valid: false},
				RegionConfig:   dbtype.RegionConfig{},
				SentinelConfig: []byte{},
				ShutdownSignal: db.AppRuntimeSettingsShutdownSignalSIGTERM,
				CreatedAt:      now,
				UpdatedAt:      sql.NullInt64{Valid: true, Int64: now},
			},
			// Worker app — preview + production
			{
				WorkspaceID:    workspaceID,
				AppID:          workerAppID,
				EnvironmentID:  previewEnvID,
				Port:           8080,
				CpuMillicores:  256,
				MemoryMib:      256,
				Command:        dbtype.StringSlice{},
				Healthcheck:    dbtype.NullHealthcheck{Healthcheck: nil, Valid: false},
				RegionConfig:   dbtype.RegionConfig{},
				SentinelConfig: []byte{},
				ShutdownSignal: db.AppRuntimeSettingsShutdownSignalSIGTERM,
				CreatedAt:      now,
				UpdatedAt:      sql.NullInt64{Valid: true, Int64: now},
			},
			{
				WorkspaceID:    workspaceID,
				AppID:          workerAppID,
				EnvironmentID:  productionEnvID,
				Port:           8080,
				CpuMillicores:  256,
				MemoryMib:      256,
				Command:        dbtype.StringSlice{},
				Healthcheck:    dbtype.NullHealthcheck{Healthcheck: nil, Valid: false},
				RegionConfig:   dbtype.RegionConfig{},
				SentinelConfig: []byte{},
				ShutdownSignal: db.AppRuntimeSettingsShutdownSignalSIGTERM,
				CreatedAt:      now,
				UpdatedAt:      sql.NullInt64{Valid: true, Int64: now},
			},
		})
		if err != nil {
			return fmt.Errorf("failed to create runtime settings: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	logger.Info("multi-app seed completed",
		"workspace", workspaceID,
		"project", projectID,
		"project_slug", projectSlug,
		"api_app", apiAppID,
		"worker_app", workerAppID,
		"preview_env", previewEnvID,
		"production_env", productionEnvID,
	)

	return nil
}
