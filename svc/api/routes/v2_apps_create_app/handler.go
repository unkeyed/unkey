package handler

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/rpc/ctrl"
	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/ctrlclient"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2AppsCreateAppRequestBody
	Response = openapi.V2AppsCreateAppResponseBody
)

type Handler struct {
	DB         db.Database
	Auditlogs  auditlogs.AuditLogService
	CtrlClient ctrl.AppServiceClient
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/apps.createApp"
}

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	project, err := db.Query.FindProjectByWorkspaceAndSlug(ctx, h.DB.RO(), db.FindProjectByWorkspaceAndSlugParams{
		WorkspaceID: principal.WorkspaceID,
		Slug:        req.ProjectSlug,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New(
				"project not found",
				fault.Code(codes.Data.Project.NotFound.URN()),
				fault.Internal("project not found"),
				fault.Public("The requested project does not exist."),
			)
		}
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve project."),
		)
	}

	err = principal.Authorize(rbac.U(
		urn.New().Workspace(principal.WorkspaceID).Project(project.ID),
		permissions.CreateApp{},
	))
	if err != nil {
		return err
	}

	res, err := h.CtrlClient.CreateApp(ctx, &ctrlv1.CreateAppRequest{
		WorkspaceId: principal.WorkspaceID,
		ProjectId:   project.ID,
		Name:        req.Name,
		Slug:        req.Slug,
	})
	if err != nil {
		if connect.CodeOf(err) == connect.CodeAlreadyExists {
			return fault.New(
				"app already exists",
				fault.Code(codes.Data.App.Duplicate.URN()),
				fault.Internal("app slug already exists in project"),
				fault.Public(fmt.Sprintf("An app with slug '%s' already exists in this project.", req.Slug)),
			)
		}
		return ctrlclient.HandleError(err, "create app")
	}

	err = h.Auditlogs.Insert(ctx, h.DB.RW(), []auditlog.AuditLog{
		{
			WorkspaceID:   principal.WorkspaceID,
			Event:         auditlog.AppCreateEvent,
			Display:       fmt.Sprintf("Created app %s", res.GetId()),
			ActorID:       principal.Subject.ID,
			ActorName:     principal.Subject.Name,
			ActorMeta:     map[string]any{},
			ActorType:     auditlog.AuditLogActor(principal.Subject.Type),
			RemoteIP:      s.Location(),
			UserAgent:     s.UserAgent(),
			CorrelationID: "",
			Resources: []auditlog.AuditLogResource{
				{
					ID:          res.GetId(),
					Type:        auditlog.AppResourceType,
					Meta:        map[string]any{"name": req.Name, "slug": req.Slug, "projectId": project.ID},
					Name:        req.Name,
					DisplayName: req.Name,
				},
			},
		},
	})
	if err != nil {
		return err
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.V2AppsCreateAppResponseData{
			AppId: res.GetId(),
		},
	})
}
