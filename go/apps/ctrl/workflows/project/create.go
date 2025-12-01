package project

import (
	"database/sql"
	"errors"
	"time"

	restate "github.com/restatedev/sdk-go"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/go/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func (s *Service) CreateProject(ctx restate.ObjectContext, req *hydrav1.CreateProjectRequest) (*hydrav1.CreateProjectResponse, error) {

	if err := assert.All(
		assert.NotEmpty(req.GetWorkspaceId()),
		assert.NotEmpty(req.GetName()),
		assert.NotEmpty(req.GetSlug()),
	); err != nil {
		return nil, restate.TerminalError(err)
	}

	workspace, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.Workspace, error) {
		found, err := db.Query.FindWorkspaceByID(ctx, s.db.RW(), req.GetWorkspaceId())
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

	k8sNamespace := workspace.K8sNamespace.String
	// This should really be in a dedicated createWorkspace call I think,
	// but this works for now
	if k8sNamespace == "" {
		k8sNamespace, err = restate.Run(ctx, func(runCtx restate.RunContext) (string, error) {
			name := uid.Nano(12)
			res, err := db.Query.UpdateWorkspaceK8sNamespace(runCtx, s.db.RW(), db.UpdateWorkspaceK8sNamespaceParams{
				ID:           workspace.ID,
				K8sNamespace: sql.NullString{Valid: true, String: name},
			})
			if err != nil {
				return "", err
			}
			affected, err := res.RowsAffected()
			if err != nil {
				return "", err
			}
			if affected != 1 {
				return "", errors.New("failed to update workspace k8s namespace")
			}
			return name, nil
		})
		if err != nil {
			return nil, err
		}
	}

	projectID, err := restate.Run(ctx, func(runCtx restate.RunContext) (string, error) {
		return uid.New(uid.ProjectPrefix), nil
	}, restate.WithName("generate project ID"))
	if err != nil {
		return nil, err
	}

	err = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.InsertProject(runCtx, s.db.RW(), db.InsertProjectParams{
			ID:               projectID,
			WorkspaceID:      workspace.ID,
			Name:             req.GetName(),
			Slug:             req.GetSlug(),
			GitRepositoryUrl: sql.NullString{Valid: req.GetGitRepository() != "", String: req.GetGitRepository()},
			DefaultBranch:    sql.NullString{Valid: true, String: "main"},
			DeleteProtection: sql.NullBool{Valid: true, Bool: true},

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

		err = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
			return db.Query.InsertEnvironment(runCtx, s.db.RW(), db.InsertEnvironmentParams{
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

		replicas := int32(1)
		if env.Slug == "production" {
			replicas = int32(3)
		}

		err = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
			return db.Query.InsertGateway(runCtx, s.db.RW(), db.InsertGatewayParams{
				ID:             gatewayID,
				WorkspaceID:    workspace.ID,
				ProjectID:      projectID,
				EnvironmentID:  environmentID,
				K8sServiceName: "TODO",
				Region:         "aws:us-east-1",
				Image:          "nginx:latest",
				Health:         db.GatewaysHealthUnknown,
				Replicas:       replicas,
				CpuMillicores:  1024,
				MemoryMib:      1024,
				CreatedAt:      time.Now().UnixMilli(),
			})
		}, restate.WithName("insert gateway"))
		if err != nil {
			return nil, err
		}

		err = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
			return s.cluster.EmitEvent(runCtx, map[string]string{"region": "aws:us-east-1"}, &ctrlv1.InfraEvent{
				Event: &ctrlv1.InfraEvent_GatewayEvent{
					GatewayEvent: &ctrlv1.GatewayEvent{
						Event: &ctrlv1.GatewayEvent_Apply{
							Apply: &ctrlv1.ApplyGateway{
								Namespace:     workspace.K8sNamespace.String,
								WorkspaceId:   workspace.ID,
								ProjectId:     projectID,
								EnvironmentId: environmentID,
								GatewayId:     gatewayID,
								Image:         "nginx:latest",
								Replicas:      uint32(replicas),
								CpuMillicores: 1024,
								MemorySizeMib: 1024,
							},
						},
					},
				},
			})
		}, restate.WithName("apply gateway"))
		if err != nil {
			return nil, err
		}
	}

	return &hydrav1.CreateProjectResponse{
		ProjectId: projectID,
	}, nil
}
