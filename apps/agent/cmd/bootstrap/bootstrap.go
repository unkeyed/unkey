package bootstrap

import (
	"context"

	"github.com/spf13/cobra"
	apisv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/apis/v1"
	authenticationv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/authentication/v1"
	workspacesv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/workspaces/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/database"

	"os"

	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/uid"
)

var (
	databaseDsn string
	workspaceId string
	tenantId    string
	apiId       string
)

func init() {

	Cmd.Flags().StringVar(&databaseDsn, "database-dsn", "", "Database connection string, falls back to DATABASE_DSN env variable")
	Cmd.Flags().StringVar(&workspaceId, "workspace-id", uid.Workspace(), "The workspace id to seed, generates a random one if not provided")
	Cmd.Flags().StringVar(&tenantId, "tenant-id", uid.New(16, "fake"), "The clerk tenant id to seed, generates a random one if not provided")
	Cmd.Flags().StringVar(&apiId, "api-id", uid.Api(), "The api id to seed, generates a random one if not provided")
}

// BootstrapCmd represents the agent command
var Cmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Bootstrap your database",
	Run: func(cmd *cobra.Command, args []string) {
		logger := logging.New(nil)

		if databaseDsn == "" {
			logger.Info().Msg("no database dsn provided, loading from DATABASE_DSN env variable")
			databaseDsn = os.Getenv("DATABASE_DSN")
		}
		if databaseDsn == "" {
			logger.Fatal().Msg("no database dsn provided")
		}
		ctx := context.Background()

		db, err := database.New(database.Config{
			PrimaryUs: databaseDsn,
			Logger:    logger,
		})

		if err != nil {
			logger.Fatal().Err(err).Msg("unable to connect to databae")
		}

		workspace := &workspacesv1.Workspace{
			WorkspaceId: workspaceId,
			Name:        "Unkey",
			TenantId:    tenantId,
			Plan:        workspacesv1.Plan_PLAN_FREE,
		}
		keyAuth := &authenticationv1.KeyAuth{
			KeyAuthId:   uid.KeyAuth(),
			WorkspaceId: workspace.WorkspaceId,
		}
		api := &apisv1.Api{
			ApiId:       apiId,
			Name:        "api.unkey.dev",
			WorkspaceId: workspace.WorkspaceId,
			IpWhitelist: []string{},
			AuthType:    apisv1.AuthType_AUTH_TYPE_KEY,
			KeyAuthId:   &keyAuth.KeyAuthId,
		}
		logger.Info().Msg("seeding workspace")
		err = db.InsertWorkspace(ctx, workspace)
		if err != nil {
			logger.Fatal().Err(err).Msg("unable to create workspace")
		}
		logger.Info().Msg("seeding keyAuth")
		err = db.InsertKeyAuth(ctx, keyAuth)
		if err != nil {
			logger.Fatal().Err(err).Msg("unable to create keyAuth")
		}
		logger.Info().Msg("seeding api")
		err = db.InsertApi(ctx, api)
		if err != nil {
			logger.Fatal().Err(err).Msg("unable to create api")
		}

		logger.Info().Str("workspaceId", workspace.WorkspaceId).Str("apiId", apiId).Str("tenantId", tenantId).Msg("done")
	},
}
