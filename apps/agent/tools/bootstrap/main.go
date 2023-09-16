package main

import (
	"context"
	"database/sql"
	"os"

	"github.com/unkeyed/unkey/apps/agent/pkg/database"
	"github.com/unkeyed/unkey/apps/agent/pkg/entities"
	"github.com/unkeyed/unkey/apps/agent/pkg/env"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/uid"
)

func main() {
	ctx := context.Background()
	logger := logging.New()
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

	workspace := entities.Workspace{
		Id:       e.String("UNKEY_WORKSPACE_ID", uid.Workspace()),
		Name:     "Unkey",
		TenantId: e.String("TENANT_ID", uid.New(16, "fake")),
		Plan:     entities.EnterprisePlan,
	}
	keyAuth := entities.KeyAuth{
		Id:          uid.KeyAuth(),
		WorkspaceId: workspace.Id,
	}
	api := entities.Api{
		Id:          e.String("UNKEY_API_ID", uid.Api()),
		Name:        "api.unkey.dev",
		WorkspaceId: workspace.Id,
		IpWhitelist: []string{},
		AuthType:    entities.AuthTypeKey,
		KeyAuthId:   keyAuth.Id,
	}

	err = db.InsertWorkspace(ctx, workspace)
	if err != nil {
		logger.Fatal().Err(err).Msg("unable to create workspace")
	}

	err = db.CreateKeyAuth(ctx, keyAuth)
	if err != nil {
		logger.Fatal().Err(err).Msg("unable to create keyAuth")
	}
	err = db.InsertApi(ctx, api)
	if err != nil {
		logger.Fatal().Err(err).Msg("unable to create api")
	}

}
