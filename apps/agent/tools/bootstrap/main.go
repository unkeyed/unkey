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
	"go.uber.org/zap"
)

func main() {
	ctx := context.Background()
	logger := logging.New()
	e := env.Env{ErrorHandler: func(err error) { logger.Error("unable to load env", zap.Error(err)) }}

	seedDb, err := sql.Open("mysql", e.String("DATABASE_DSN"))
	if err != nil {
		logger.Fatal("error opening database", zap.Error(err))
	}
	schema, err := os.ReadFile("../../pkg/database/schema.sql")
	if err != nil {
		logger.Fatal("error reading schema", zap.Error(err))
	}
	_, err = seedDb.Exec(string(schema))
	if err != nil {
		logger.Fatal("error pushing schema", zap.Error(err))
	}
	err = seedDb.Close()
	if err != nil {
		logger.Fatal("uanble to close seed db", zap.Error(err))
	}

	db, err := database.New(database.Config{
		PrimaryUs: e.String("DATABASE_DSN"),
		Logger:    logger,
	})

	if err != nil {
		logger.Fatal("unable to connect to databae", zap.Error(err))
	}

	workspace := entities.Workspace{
		Id:       e.String("UNKEY_WORKSPACE_ID", uid.Workspace()),
		Name:     "Unkey",
		Slug:     "unkey",
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
		logger.Fatal("unable to create workspace", zap.Error(err))
	}

	err = db.CreateKeyAuth(ctx, keyAuth)
	if err != nil {
		logger.Fatal("unable to create keyAuth", zap.Error(err))
	}
	err = db.InsertApi(ctx, api)
	if err != nil {
		logger.Fatal("unable to create api", zap.Error(err))
	}

}
