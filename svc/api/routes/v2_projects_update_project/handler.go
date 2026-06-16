package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2ProjectsUpdateProjectRequestBody
	Response = openapi.V2ProjectsUpdateProjectResponseBody
)

type Handler struct {
	DB        db.Database
	Auditlogs auditlogs.AuditLogService
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/projects.updateProject"
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
		Slug:        req.Slug,
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

	err = principal.Authorize(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Project,
			ResourceID:   "*",
			Action:       rbac.UpdateProject,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Project,
			ResourceID:   project.ID,
			Action:       rbac.UpdateProject,
		}),
	))
	if err != nil {
		return err
	}

	if req.Name == nil && req.DeleteProtection == nil {
		return s.JSON(http.StatusOK, Response{
			Meta: openapi.Meta{
				RequestId: s.RequestID(),
			},
			Data: openapi.Project{
				Id:               project.ID,
				Name:             project.Name,
				Slug:             project.Slug,
				CreatedAt:        project.CreatedAt,
				UpdatedAt:        project.UpdatedAt.Int64,
				DeleteProtection: project.DeleteProtection.Bool,
			},
		})
	}

	name := project.Name
	if req.Name != nil {
		name = *req.Name
	}

	deleteProtection := project.DeleteProtection.Bool
	if req.DeleteProtection != nil {
		deleteProtection = *req.DeleteProtection
	}

	updatedAt := time.Now().UnixMilli()

	err = db.TxRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		err = db.Query.UpdateProject(ctx, tx, db.UpdateProjectParams{
			WorkspaceID:      principal.WorkspaceID,
			Slug:             req.Slug,
			Name:             name,
			DeleteProtection: sql.NullBool{Valid: true, Bool: deleteProtection},
			UpdatedAt:        sql.NullInt64{Valid: true, Int64: updatedAt},
		})
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("unable to update project"),
				fault.Public("We're unable to update the project."),
			)
		}

		err = h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
			{
				WorkspaceID:   principal.WorkspaceID,
				Event:         auditlog.ProjectUpdateEvent,
				Display:       fmt.Sprintf("Updated project %s", project.ID),
				ActorID:       principal.Subject.ID,
				ActorName:     principal.Subject.Name,
				ActorMeta:     map[string]any{},
				ActorType:     auditlog.AuditLogActor(principal.Subject.Type),
				RemoteIP:      s.Location(),
				UserAgent:     s.UserAgent(),
				CorrelationID: "",
				Resources: []auditlog.AuditLogResource{
					{
						ID:          project.ID,
						Type:        auditlog.ProjectResourceType,
						Meta:        map[string]any{"name": name, "slug": project.Slug, "deleteProtection": deleteProtection},
						Name:        name,
						DisplayName: name,
					},
				},
			},
		})
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.Project{
			Id:               project.ID,
			Name:             name,
			Slug:             project.Slug,
			CreatedAt:        project.CreatedAt,
			UpdatedAt:        updatedAt,
			DeleteProtection: deleteProtection,
		},
	})
}
