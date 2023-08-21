package database

import (
	"context"
	"errors"
	"github.com/go-sql-driver/mysql"

	gen "github.com/unkeyed/unkey/apps/agent/gen/database"
	"github.com/unkeyed/unkey/apps/agent/pkg/entities"
)

func (db *database) InsertWorkspace(ctx context.Context, ws entities.Workspace) error {

	err := db.write().InsertWorkspace(ctx, gen.InsertWorkspaceParams{
		ID:       ws.Id,
		Name:     ws.Name,
		Slug:     ws.Slug,
		TenantID: ws.TenantId,
		Plan:     gen.NullWorkspacesPlan{WorkspacesPlan: gen.WorkspacesPlan(ws.Plan), Valid: ws.Plan != ""},
	})
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) {
			switch mysqlErr.Number {
			case 1062:
				return ErrNotUnique
			}
		}
		return err
	}
	return nil

}
