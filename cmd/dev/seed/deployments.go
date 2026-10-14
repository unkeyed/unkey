package seed

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/db"
	dbtype "github.com/unkeyed/unkey/pkg/db/types"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/uid"
)

var deploymentsCmd = &cli.Command{
	Name:  "deployments",
	Usage: "Seed ready deployments into one app environment, one minute apart",
	Flags: []cli.Flag{
		cli.String("workspace", "Workspace ID to seed", cli.Default("ws_local")),
		cli.String("project", "Project ID or slug to seed", cli.Default("local-api")),
		cli.String("app", "App ID or slug to seed", cli.Default("default")),
		cli.String("environment", "Environment slug to seed", cli.Default("preview")),
		cli.Int("count", "Number of deployments to create", cli.Default(30)),
		cli.String("database-primary", "MySQL database DSN", cli.Default("unkey:password@tcp(127.0.0.1:3306)/unkey?parseTime=true&interpolateParams=true"), cli.EnvVar("UNKEY_DATABASE_PRIMARY")),
	},
	Action: seedDeployments,
}

func seedDeployments(ctx context.Context, cmd *cli.Command) error {
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
	app, err := db.Query.FindAppByProjectAndIdOrSlug(ctx, database.RO(), db.FindAppByProjectAndIdOrSlugParams{
		WorkspaceID: workspaceID,
		Project:     cmd.RequireString("project"),
		App:         cmd.RequireString("app"),
	})
	if err != nil {
		return fmt.Errorf("failed to find app: %w", err)
	}
	environment, err := db.Query.FindEnvironmentByAppIdAndSlug(ctx, database.RO(), db.FindEnvironmentByAppIdAndSlugParams{
		AppID: app.ID,
		Slug:  cmd.RequireString("environment"),
	})
	if err != nil {
		return fmt.Errorf("failed to find environment: %w", err)
	}

	count := cmd.Int("count")
	now := time.Now()
	err = db.Tx(ctx, database.RW(), func(ctx context.Context, tx db.DBTX) error {
		for i := range count {
			id := uid.New(uid.DeploymentPrefix)
			createdAt := now.Add(-time.Duration(i) * time.Minute).UnixMilli()
			err := db.Query.InsertDeployment(ctx, tx, db.InsertDeploymentParams{
				ID:                            id,
				K8sName:                       "seed-deployments-" + id,
				WorkspaceID:                   workspaceID,
				ProjectID:                     app.ProjectID,
				AppID:                         app.ID,
				EnvironmentID:                 environment.ID,
				Source:                        db.DeploymentsSourceGit,
				ImageRequested:                sql.NullString{},
				GitCommitSha:                  sql.NullString{String: fmt.Sprintf("%040x", createdAt), Valid: true},
				GitBranch:                     sql.NullString{String: "main", Valid: true},
				SentinelConfig:                []byte("{}"),
				GitCommitMessage:              sql.NullString{String: fmt.Sprintf("Seeded deployment %d", count-i), Valid: true},
				GitCommitAuthorHandle:         sql.NullString{String: "deployments-seed", Valid: true},
				GitCommitAuthorAvatarUrl:      sql.NullString{},
				GitCommitTimestamp:            sql.NullInt64{Int64: createdAt, Valid: true},
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
				TriggeredBy:                   sql.NullString{String: "deployments-seed", Valid: true},
				TriggerReason:                 sql.NullString{},
				CreatedAt:                     createdAt,
				UpdatedAt:                     sql.NullInt64{},
			})
			if err != nil {
				return fmt.Errorf("failed to create deployment %s: %w", id, err)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	logger.Info("seeded deployments",
		"workspace_id", workspaceID,
		"project_id", app.ProjectID,
		"app_id", app.ID,
		"environment_id", environment.ID,
		"count", count,
	)
	return nil
}
