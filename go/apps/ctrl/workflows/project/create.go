package project

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

		var ws db.Workspace
		err := db.Tx(runCtx, s.db.RW(), func(txCtx context.Context, tx db.DBTX) error {

			found, err := db.Query.FindWorkspaceByID(txCtx, tx, req.GetWorkspaceId())
			if err != nil {
				if db.IsNotFound(err) {
					return restate.TerminalError(errors.New("workspace not found"))
				}
				return err
			}
			ws = found

			if !found.K8sNamespace.Valid {
				ws.K8sNamespace.Valid = true
				ws.K8sNamespace.String = uid.Nano(8)
				return db.Query.SetWorkspaceK8sNamespace(txCtx, tx, db.SetWorkspaceK8sNamespaceParams{
					ID:           ws.ID,
					K8sNamespace: ws.K8sNamespace,
				})
			}
			ws = found

			return nil
		})
		return ws, err
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
		{Slug: "preview", Description: "Preview environment"},
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
				ID:             environmentID,
				WorkspaceID:    workspace.ID,
				ProjectID:      projectID,
				Slug:           env.Slug,
				Description:    env.Description,
				CreatedAt:      time.Now().UnixMilli(),
				UpdatedAt:      sql.NullInt64{Valid: false, Int64: 0},
				SentinelConfig: []byte(""),
			})
		}, restate.WithName("insert environment"))

		if err != nil {
			return nil, err
		}
		sentinelID, err := restate.Run(ctx, func(runCtx restate.RunContext) (string, error) {
			return uid.New(uid.SentinelPrefix), nil
		})
		if err != nil {
			return nil, err
		}

		replicas := int32(1)
		if env.Slug == "production" {
			replicas = int32(3)
		}

		k8sCrdName := fmt.Sprintf("gw-%s", uid.NanoLower(8))

		err = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
			return db.Query.InsertSentinel(runCtx, s.db.RW(), db.InsertSentinelParams{
				ID:              sentinelID,
				WorkspaceID:     workspace.ID,
				ProjectID:       projectID,
				EnvironmentID:   environmentID,
				K8sServiceName:  "TODO",
				K8sCrdName:      k8sCrdName,
				Region:          "aws:us-east-1",
				Image:           s.sentinelImage,
				Health:          db.SentinelsHealthUnknown,
				DesiredReplicas: replicas,
				Replicas:        0,
				CpuMillicores:   256,
				MemoryMib:       256,
				CreatedAt:       time.Now().UnixMilli(),
			})
		}, restate.WithName("insert sentinel"))
		if err != nil {
			return nil, err
		}

		err = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
			return s.cluster.EmitEvent(runCtx, map[string]string{"region": "aws:us-east-1", "shard": "default"}, &ctrlv1.InfraEvent{
				Event: &ctrlv1.InfraEvent_SentinelEvent{
					SentinelEvent: &ctrlv1.SentinelEvent{
						Event: &ctrlv1.SentinelEvent_Apply{
							Apply: &ctrlv1.ApplySentinel{
								// already ensured to exist above
								Namespace:     workspace.K8sNamespace.String,
								K8SCrdName:    k8sCrdName,
								WorkspaceId:   workspace.ID,
								ProjectId:     projectID,
								EnvironmentId: environmentID,
								SentinelId:    sentinelID,
								Image:         s.sentinelImage,
								Replicas:      uint32(replicas),
								CpuMillicores: 256,
								MemorySizeMib: 256,
							},
						},
					},
				},
			})
		}, restate.WithName(fmt.Sprintf("apply sentinel %s", sentinelID)))
		if err != nil {
			return nil, err
		}
	}

	return &hydrav1.CreateProjectResponse{
		ProjectId: projectID,
	}, nil
}
