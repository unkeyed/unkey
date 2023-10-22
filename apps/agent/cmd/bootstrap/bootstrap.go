package bootstrap

import (
	"context"
	"github.com/spf13/cobra"
	"github.com/unkeyed/unkey/apps/agent/pkg/database"
	"github.com/unkeyed/unkey/apps/agent/pkg/entities"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/uid"
	"os"
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

		workspace := entities.Workspace{
			Id:       workspaceId,
			Name:     "Unkey",
			TenantId: tenantId,
			Plan:     entities.EnterprisePlan,
		}
		keyAuth := entities.KeyAuth{
			Id:          uid.KeyAuth(),
			WorkspaceId: workspace.Id,
		}
		api := entities.Api{
			Id:          apiId,
			Name:        "api.unkey.dev",
			WorkspaceId: workspace.Id,
			IpWhitelist: []string{},
			AuthType:    entities.AuthTypeKey,
			KeyAuthId:   keyAuth.Id,
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

		logger.Info().Str("workspaceId", workspace.Id).Str("apiId", apiId).Str("tenantId", tenantId).Msg("done")
	},
}
