package project

import (
	"database/sql"
	"errors"
	"time"

	"connectrpc.com/connect"
	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/go/gen/proto/hydra/v1"
	kranev1 "github.com/unkeyed/unkey/go/gen/proto/krane/v1"
	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func (s *Service) CreateProject(ctx restate.ObjectContext, req *hydrav1.CreateProjectRequest) (*hydrav1.CreateProjectResponse, error) {

	if err := assert.All(
		assert.NotEmpty(req.WorkspaceId),
		assert.NotEmpty(req.Name),
		assert.NotEmpty(req.Slug),
	); err != nil {
		return nil, restate.TerminalError(err)
	}

	workspace, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.Workspace, error) {
		found, err := db.Query.FindWorkspaceByID(ctx, s.db.RW(), req.WorkspaceId)
		if err != nil {
			if db.IsNotFound(err) {
				return db.Workspace{}, restate.TerminalError(errors.New("workspace not found"))
			}
			return db.Workspace{}, err
		}
		return found, nil
	}, restate.WithName("find workspace"))

	if err != nil {
		return nil, err
	}

	projectID, err := restate.Run(ctx, func(runCtx restate.RunContext) (string, error) {
		return uid.New(uid.ProjectPrefix), nil
	}, restate.WithName("generate project ID"))
	if err != nil {
		return nil, err
	}

	_, err = restate.Run(ctx, func(runCtx restate.RunContext) (restate.Void, error) {
		return restate.Void{}, db.Query.InsertProject(runCtx, s.db.RW(), db.InsertProjectParams{
			ID:               projectID,
			WorkspaceID:      workspace.ID,
			Name:             req.Name,
			Slug:             req.Slug,
			GitRepositoryUrl: sql.NullString{Valid: req.GitRepository != "", String: req.GitRepository},
			DefaultBranch:    sql.NullString{Valid: true, String: "main"},

			CreatedAt: time.Now().UnixMilli(),
			UpdatedAt: sql.NullInt64{Valid: false, Int64: 0},
		})
	}, restate.WithName("insert project"))

	if err != nil {
		return nil, err
	}

	environments := []struct {
		Slug        string
		Description string
	}{
		{Slug: "development", Description: "Development environment"},
		{Slug: "production", Description: "Production environment"},
	}

	for _, env := range environments {
		environmentID, err := restate.Run(ctx, func(runCtx restate.RunContext) (string, error) {
			return uid.New(uid.EnvironmentPrefix), nil
		}, restate.WithName("create environment id"))

		if err != nil {
			return nil, err
		}

		_, err = restate.Run(ctx, func(runCtx restate.RunContext) (restate.Void, error) {
			return restate.Void{}, db.Query.InsertEnvironment(runCtx, s.db.RW(), db.InsertEnvironmentParams{
				ID:            environmentID,
				WorkspaceID:   workspace.ID,
				ProjectID:     projectID,
				Slug:          env.Slug,
				Description:   env.Description,
				CreatedAt:     time.Now().UnixMilli(),
				UpdatedAt:     sql.NullInt64{Valid: false, Int64: 0},
				GatewayConfig: []byte(""),
			})
		}, restate.WithName("insert environment"))

		if err != nil {
			return nil, err
		}
		gatewayID, err := restate.Run(ctx, func(runCtx restate.RunContext) (string, error) {
			return uid.New(uid.GatewayPrefix), nil
		})
		if err != nil {
			return nil, err
		}

		replicas := uint32(1)
		if env.Slug == "production" {
			replicas = uint32(3)
		}

		_, err = restate.Run(ctx, func(runCtx restate.RunContext) (restate.Void, error) {
			_, err := s.krane.CreateGateway(runCtx, connect.NewRequest(&kranev1.CreateGatewayRequest{
				Gateway: &kranev1.GatewayRequest{
					Namespace:     workspace.ID,
					WorkspaceId:   workspace.ID,
					GatewayId:     gatewayID,
					Image:         "nginx:latest", // TODO
					Replicas:      replicas,
					CpuMillicores: uint32(128),
					MemorySizeMib: uint64(256),
				},
			}))
			return restate.Void{}, err
		}, restate.WithName("provision gateway"))

		if err != nil {
			return nil, err
		}

	}

	return &hydrav1.CreateProjectResponse{
		ProjectId: projectID,
	}, nil
}
