package database

import (
	"context"
	"fmt"

	"github.com/chronark/unkey/apps/api/pkg/entities"
)

func (db *Database) CreateWorkspace(ctx context.Context, newWorkspace entities.Workspace) error {
	ctx, span := db.tracer.Start(ctx, "db.createWorkspace")
	defer span.End()
	workpspace := workspaceEntityToModel(newWorkspace)

	err := workpspace.Insert(ctx, db.write())
	if err != nil {
		return fmt.Errorf("unable to insert key, %w", err)
	}
	return nil
}
