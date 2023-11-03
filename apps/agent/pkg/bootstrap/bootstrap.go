package bootstrap

import (
	"bufio"
	"context"
	"fmt"

	"os"

	apisv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/apis/v1"
	authenticationv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/authentication/v1"
	workspacesv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/workspaces/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/database"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/uid"
)

type BootstrapResponse struct {
	StaticKey string
}

func BootstrapAgent(ctx context.Context, databaseDsn string, outFile string) (BootstrapResponse, error) {
	if databaseDsn == "" {
		return BootstrapResponse{}, fmt.Errorf("database dsn is required")
	}

	db, err := database.New(database.Config{
		PrimaryUs: databaseDsn,
		Logger:    logging.NewNoop(),
	})

	if err != nil {
		return BootstrapResponse{}, fmt.Errorf("unable to connect to database: %w", err)
	}

	// Seeding unkey stuff

	workspace := &workspacesv1.Workspace{
		WorkspaceId: uid.Workspace(),
		Name:        uid.New(8, "boot"),
		TenantId:    uid.New(8, "boot"),
		Plan:        workspacesv1.Plan_PLAN_ENTERPRISE,
	}
	keyAuth := &authenticationv1.KeyAuth{
		KeyAuthId:   uid.KeyAuth(),
		WorkspaceId: workspace.WorkspaceId,
	}
	api := &apisv1.Api{
		ApiId:       uid.Api(),
		Name:        "api.unkey.dev",
		WorkspaceId: workspace.WorkspaceId,
		IpWhitelist: []string{},
		AuthType:    apisv1.AuthType_AUTH_TYPE_KEY,
		KeyAuthId:   &keyAuth.KeyAuthId,
	}
	err = db.InsertWorkspace(ctx, workspace)
	if err != nil {
		return BootstrapResponse{}, fmt.Errorf("unable to create unkey workspace: %w", err)
	}
	err = db.InsertKeyAuth(ctx, keyAuth)
	if err != nil {
		return BootstrapResponse{}, fmt.Errorf("unable to create unkey key auth: %w", err)
	}
	err = db.InsertApi(ctx, api)
	if err != nil {
		return BootstrapResponse{}, fmt.Errorf("unable to create unkey api: %w", err)
	}

	appAuthToken := uid.New(32, "unkey_static")
	if outFile != "" {
		file, err := os.OpenFile(outFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return BootstrapResponse{}, fmt.Errorf("unable to open output file: %w", err)
		}
		defer file.Close()

		lines := []string{
			fmt.Sprintf("UNKEY_WORKSPACE_ID=\"%s\"", workspace.WorkspaceId),
			fmt.Sprintf("UNKEY_API_ID=\"%s\"", api.ApiId),
			fmt.Sprintf("UNKEY_KEY_AUTH_ID=\"%s\"", keyAuth.KeyAuthId),
			fmt.Sprintf("UNKEY_APP_AUTH_TOKEN=\"%s\"", appAuthToken),
		}

		buf := bufio.NewWriter(file)
		for _, ln := range lines {
			_, err = fmt.Fprintln(buf, ln)
			if err != nil {
				return BootstrapResponse{}, fmt.Errorf("unable to write to output file: %w", err)
			}
		}
		err = buf.Flush()
		if err != nil {
			return BootstrapResponse{}, fmt.Errorf("unable to flush output file: %w", err)
		}
	}

	return BootstrapResponse{
		StaticKey: appAuthToken,
	}, nil
}
