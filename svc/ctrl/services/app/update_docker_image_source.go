package app

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/deploy/imageref"
	"github.com/unkeyed/unkey/svc/ctrl/internal/actor"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func (s *Service) UpdateDockerImageSource(
	ctx context.Context,
	req *connect.Request[ctrlv1.UpdateDockerImageSourceRequest],
) (*connect.Response[ctrlv1.UpdateDockerImageSourceResponse], error) {
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return nil, err
	}
	if err := assert.All(
		assert.NotEmpty(req.Msg.GetWorkspaceId(), "workspace_id is required"),
		assert.NotEmpty(req.Msg.GetAppId(), "app_id is required"),
		assert.NotEmpty(req.Msg.GetImageReference(), "image_reference is required"),
		assert.NotNil(req.Msg.GetActor(), "actor is required"),
	); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	imageReference, err := imageref.Normalize(req.Msg.GetImageReference())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	app, err := s.db.FindAppById(ctx, req.Msg.GetAppId())
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("app %q not found", req.Msg.GetAppId()))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("find app: %w", err))
	}
	if app.WorkspaceID != req.Msg.GetWorkspaceId() {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("app %q not found", req.Msg.GetAppId()))
	}
	if app.SourceType != db.AppsSourceTypeDockerImage {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("app %q is not Docker-sourced", app.ID))
	}

	currentSource, err := s.db.FindAppDockerSourceByAppId(ctx, app.ID)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeFailedPrecondition,
				fmt.Errorf("Docker-sourced app %q has no source configuration", app.ID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("find Docker source: %w", err))
	}

	now := time.Now().UnixMilli()
	err = db.TxRetry(ctx, s.db.RW(), func(txCtx context.Context, tx db.DBTX) error {
		queries := db.NewQueries(tx)
		if txErr := queries.UpdateAppDockerSourceImageReference(txCtx, db.UpdateAppDockerSourceImageReferenceParams{
			ImageReference: imageReference,
			UpdatedAt:      sql.NullInt64{Valid: true, Int64: now},
			AppID:          app.ID,
			WorkspaceID:    app.WorkspaceID,
		}); txErr != nil {
			return fmt.Errorf("update Docker source: %w", txErr)
		}

		a := req.Msg.GetActor()
		if txErr := s.auditlogs.Insert(txCtx, tx, []auditlog.AuditLog{
			{
				WorkspaceID:   app.WorkspaceID,
				Event:         auditlog.AppUpdateEvent,
				Display:       fmt.Sprintf("Updated Docker image source for app %s", app.ID),
				ActorID:       a.GetId(),
				ActorName:     a.GetName(),
				ActorType:     actor.AuditType(a.GetType()),
				ActorMeta:     actor.Meta(a.GetMeta()),
				RemoteIP:      a.GetRemoteIp(),
				UserAgent:     a.GetUserAgent(),
				CorrelationID: "",
				Resources: []auditlog.AuditLogResource{
					{
						ID:          app.ID,
						Type:        auditlog.AppResourceType,
						Name:        app.Name,
						DisplayName: app.Name,
						Meta: map[string]any{
							"previousImageReference": currentSource.ImageReference,
							"imageReference":         imageReference,
						},
					},
				},
			},
		}); txErr != nil {
			return fmt.Errorf("insert audit log: %w", txErr)
		}
		return nil
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&ctrlv1.UpdateDockerImageSourceResponse{
		ImageReference: imageReference,
	}), nil
}
