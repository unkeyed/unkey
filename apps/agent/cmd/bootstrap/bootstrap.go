package bootstrap

import (
	"bufio"
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	apisv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/apis/v1"
	authenticationv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/authentication/v1"
	workspacesv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/workspaces/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/database"
	"github.com/unkeyed/unkey/apps/agent/pkg/hash"
	"github.com/unkeyed/unkey/apps/agent/pkg/services/keys/keygen"

	"os"

	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/uid"
)

var (
	databaseDsn string
	workspaceId string
	tenantId    string
	apiId       string
	outputFile  string
)

func init() {

	Cmd.Flags().StringVar(&databaseDsn, "database-dsn", "", "Database connection string, falls back to DATABASE_DSN env variable")
	Cmd.Flags().StringVar(&workspaceId, "workspace-id", uid.Workspace(), "The workspace id to seed, generates a random one if not provided")
	Cmd.Flags().StringVar(&tenantId, "tenant-id", uid.New(16, "fake"), "The clerk tenant id to seed, generates a random one if not provided")
	Cmd.Flags().StringVar(&apiId, "api-id", uid.Api(), "The api id to seed, generates a random one if not provided")
	Cmd.Flags().StringVar(&outputFile, "out", "", "The file to write the generated config to, if not provided will write to stdout")
}

// BootstrapCmd represents the agent command
var Cmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Bootstrap your database",
	Run: func(cmd *cobra.Command, args []string) {
		logger := logging.New(nil)

		if databaseDsn == "" {
			logger.Info().Msg("looking for DATABASE_DSN env variable")
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

		// Seeding unkey stuff

		unkeyWorkspace := &workspacesv1.Workspace{
			WorkspaceId: uid.Workspace(),
			Name:        uid.New(8, "test"),
			TenantId:    uid.New(8, "test"),
			Plan:        workspacesv1.Plan_PLAN_FREE,
		}
		unkeyKeyAuth := &authenticationv1.KeyAuth{
			KeyAuthId:   uid.KeyAuth(),
			WorkspaceId: unkeyWorkspace.WorkspaceId,
		}
		unkeyApi := &apisv1.Api{
			ApiId:       uid.Api(),
			Name:        "api.unkey.dev",
			WorkspaceId: unkeyWorkspace.WorkspaceId,
			IpWhitelist: []string{},
			AuthType:    apisv1.AuthType_AUTH_TYPE_KEY,
			KeyAuthId:   &unkeyKeyAuth.KeyAuthId,
		}
		err = db.InsertWorkspace(ctx, unkeyWorkspace)
		if err != nil {
			logger.Fatal().Err(err).Msg("unable to create unkey workspace")
		}
		err = db.InsertKeyAuth(ctx, unkeyKeyAuth)
		if err != nil {
			logger.Fatal().Err(err).Msg("unable to create unkey keyAuth")
		}
		err = db.InsertApi(ctx, unkeyApi)
		if err != nil {
			logger.Fatal().Err(err).Msg("unable to create unkey api")
		}
		//seeding user stuff

		userWorkspace := &workspacesv1.Workspace{
			WorkspaceId: workspaceId,
			Name:        "User",
			TenantId:    tenantId,
			Plan:        workspacesv1.Plan_PLAN_FREE,
		}
		userKeyAuth := &authenticationv1.KeyAuth{
			KeyAuthId:   uid.KeyAuth(),
			WorkspaceId: userWorkspace.WorkspaceId,
		}
		userApi := &apisv1.Api{
			ApiId:       apiId,
			Name:        "User",
			WorkspaceId: userWorkspace.WorkspaceId,
			IpWhitelist: []string{},
			AuthType:    apisv1.AuthType_AUTH_TYPE_KEY,
			KeyAuthId:   &userKeyAuth.KeyAuthId,
		}
		rootKey, err := keygen.NewV1Key("unkey", 16)
		if err != nil {
			logger.Fatal().Err(err).Msg("unable to generate root key")
		}
		userRootKey := &authenticationv1.Key{
			KeyId:          uid.Key(),
			KeyAuthId:      userKeyAuth.KeyAuthId,
			WorkspaceId:    unkeyWorkspace.WorkspaceId,
			ForWorkspaceId: &userWorkspace.WorkspaceId,
			Hash:           hash.Sha256(rootKey),
			CreatedAt:      time.Now().UnixMilli(),
			Start:          rootKey[:5],
		}
		err = db.InsertWorkspace(ctx, userWorkspace)
		if err != nil {
			logger.Fatal().Err(err).Msg("unable to create user workspace")
		}
		err = db.InsertKeyAuth(ctx, userKeyAuth)
		if err != nil {
			logger.Fatal().Err(err).Msg("unable to create user keyAuth")
		}
		err = db.InsertApi(ctx, userApi)
		if err != nil {
			logger.Fatal().Err(err).Msg("unable to create user api")
		}
		err = db.InsertKey(ctx, userRootKey)
		if err != nil {
			logger.Fatal().Err(err).Msg("unable to create rootKey")
		}

		if outputFile == "" {
			logger.Info().Str("workspaceId", workspaceId).Str("apiId", apiId).Str("tenantId", tenantId).Str("rootKey", rootKey).Msg("done")
		} else {
			logger.Info().Msgf("writing environment variables to %s", outputFile)
			file, err := os.OpenFile(outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				logger.Fatal().Err(err).Msg("unable to open output file")
			}
			defer file.Close()

			lines := []string{
				fmt.Sprintf("UNKEY_WORKSPACE_ID=\"%s\"", userWorkspace.WorkspaceId),
				fmt.Sprintf("UNKEY_API_ID=\"%s\"", userApi.ApiId),
				fmt.Sprintf("UNKEY_ROOT_KEY=\"%s\"", rootKey),
			}

			buf := bufio.NewWriter(file)
			for _, ln := range lines {
				_, err = fmt.Fprintln(buf, ln)
				if err != nil {
					logger.Fatal().Err(err).Msg("unable to write line to buffer")
				}
			}
			err = buf.Flush()
			if err != nil {
				logger.Fatal().Err(err).Msg("unable to write to output file")
			}
		}
	},
}
