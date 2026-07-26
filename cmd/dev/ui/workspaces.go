package ui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/tui/app"
)

type workspaceRow struct {
	ID               string
	Slug             string
	Name             string
	StripeCustomerID string
}

type workspacesMsg struct {
	rows []workspaceRow
	err  error
}

func loadWorkspacesCmd() app.Cmd {
	return func() app.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		rows, err := listWorkspaces(ctx, databaseDSN())
		return workspacesMsg{rows: rows, err: err}
	}
}

func listWorkspaces(ctx context.Context, dsn string) ([]workspaceRow, error) {
	if dsn == "" {
		dsn = databaseDSN()
	}
	database, err := db.New(db.Config{
		PrimaryDSN:  dsn,
		ReadOnlyDSN: dsn,
		Tags:        sqlcomment.Disabled(),
	})
	if err != nil {
		return nil, fmt.Errorf("mysql: %w", err)
	}
	dbRows, err := db.Query.ListWorkspaces(ctx, database.RO(), "")
	if err != nil {
		return nil, err
	}
	out := make([]workspaceRow, 0, len(dbRows))
	for _, row := range dbRows {
		stripe := ""
		if row.Workspace.StripeCustomerID.Valid {
			stripe = row.Workspace.StripeCustomerID.String
		}
		out = append(out, workspaceRow{
			ID:               row.Workspace.ID,
			Slug:             row.Workspace.Slug,
			Name:             row.Workspace.Name,
			StripeCustomerID: stripe,
		})
	}
	return out, nil
}

// listAPIs returns the APIs in a workspace as pickable choices. apis carry
// workspace_id directly (no project layer) and use the _m timestamp columns.
func listAPIs(ctx context.Context, workspaceID string) ([]namedChoice, error) {
	return queryNamedChoices(ctx,
		"SELECT id, name FROM apis WHERE workspace_id = ? AND deleted_at_m IS NULL ORDER BY created_at_m DESC",
		workspaceID,
	)
}

type apiChoicesMsg struct {
	workspaceID string
	choices     []namedChoice
	err         error
}

func listAPIsCmd(ws workspaceRow) app.Cmd {
	return func() app.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		choices, err := listAPIs(ctx, ws.ID)
		return apiChoicesMsg{workspaceID: ws.ID, choices: choices, err: err}
	}
}

func readActiveWorkspaceID() string {
	data, err := os.ReadFile("dev/.env.seed")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "UNKEY_WORKSPACE_ID=") {
			return strings.TrimPrefix(line, "UNKEY_WORKSPACE_ID=")
		}
	}
	return ""
}
