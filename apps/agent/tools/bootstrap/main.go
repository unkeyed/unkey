package main

import (
	"context"
	"database/sql"
	"os"

	apisv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/apis/v1"
	authenticationv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/authentication/v1"
	workspacesv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/workspaces/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/database"

	"github.com/unkeyed/unkey/apps/agent/pkg/env"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/uid"
)

func main() {
	ctx := context.Background()
	logger := logging.New(nil)

	e := env.Env{ErrorHandler: func(err error) { logger.Err(err).Msg("unable to load env") }}

	seedDb, err := sql.Open("mysql", e.String("DATABASE_DSN"))
	if err != nil {
		logger.Fatal().Err(err).Msg("error opening database")
	}
	schema, err := os.ReadFile("../../pkg/database/schema.sql")
	if err != nil {
		logger.Fatal().Err(err).Msg("error reading schema")
	}
	_, err = seedDb.Exec(string(schema))
	if err != nil {
		logger.Fatal().Err(err).Msg("error pushing schema")
	}
	err = seedDb.Close()
	if err != nil {
		logger.Fatal().Err(err).Msg("uanble to close seed db")
	}

	db, err := database.New(database.Config{
		PrimaryUs: e.String("DATABASE_DSN"),
		Logger:    logger,
	})

	if err != nil {
		logger.Fatal().Err(err).Msg("unable to connect to databae")
	}

	workspace := &workspacesv1.Workspace{
		WorkspaceId: e.String("UNKEY_WORKSPACE_ID", uid.Workspace()),
		Name:        "Unkey",
		TenantId:    e.String("TENANT_ID", uid.New(16, "fake")),
		Plan:        workspacesv1.Plan_PLAN_ENTERPRISE,
	}
	keyAuth := &authenticationv1.KeyAuth{
		KeyAuthId:   uid.KeyAuth(),
		WorkspaceId: workspace.WorkspaceId,
	}
	api := &apisv1.Api{
		ApiId:       e.String("UNKEY_API_ID", uid.Api()),
		Name:        "api.unkey.dev",
		WorkspaceId: workspace.WorkspaceId,
		IpWhitelist: []string{},
		AuthType:    apisv1.AuthType_AUTH_TYPE_KEY,
		KeyAuthId:   &keyAuth.KeyAuthId,
	}

	err = db.InsertWorkspace(ctx, workspace)
	if err != nil {
		logger.Fatal().Err(err).Msg("unable to create workspace")
	}

	err = db.InsertKeyAuth(ctx, keyAuth)
	if err != nil {
		logger.Fatal().Err(err).Msg("unable to create keyAuth")
	}
	err = db.InsertApi(ctx, api)
	if err != nil {
		logger.Fatal().Err(err).Msg("unable to create api")
	}

}
