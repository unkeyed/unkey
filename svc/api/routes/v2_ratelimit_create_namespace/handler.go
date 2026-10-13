package handler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/projects"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2RatelimitCreateNamespaceRequestBody
	Response = openapi.V2RatelimitCreateNamespaceResponseBody
)

type Handler struct {
	DB        db.Database
	Auditlogs auditlogs.AuditLogService
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/ratelimit.createNamespace"
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

	namespace := openapi.RatelimitNamespace{
		Id:        uid.New(uid.RatelimitNamespacePrefix),
		Name:      req.Name,
		CreatedAt: time.Now().UnixMilli(),
		UpdatedAt: 0,
	}

	err = db.TxRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		projectID, err := projects.EnsureDefaultProject(ctx, tx, principal.AuthorizedWorkspaceID)
		if err != nil {
			return err
		}

		err = principal.Authorize(rbac.U(
			urn.New().Workspace(principal.AuthorizedWorkspaceID).Project(projectID).RatelimitNamespace("*"),
			permissions.Write,
		))
		if err != nil {
			return err
		}

		err = db.Query.InsertRatelimitNamespace(ctx, tx, db.InsertRatelimitNamespaceParams{
			ID:          namespace.Id,
			WorkspaceID: principal.AuthorizedWorkspaceID,
			ProjectID:   projectID,
			Name:        namespace.Name,
			CreatedAt:   namespace.CreatedAt,
		})
		if db.IsDuplicateKeyError(err) {
			return fault.Wrap(err,
				fault.Code(codes.Data.RatelimitNamespace.Duplicate.URN()),
				fault.Internal("namespace name already exists"),
				fault.Public(fmt.Sprintf("A namespace named '%s' already exists or was deleted in this workspace. Choose a different name.", namespace.Name)),
			)
		}
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"),
				fault.Public("Failed to create namespace."),
			)
		}

		return h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
			{
				WorkspaceID:   principal.AuthorizedWorkspaceID,
				Event:         auditlog.RatelimitNamespaceCreateEvent,
				ActorType:     auditlog.AuditLogActor(principal.Subject.Type),
				ActorID:       principal.Subject.ID,
				ActorName:     principal.Subject.Name,
				ActorMeta:     map[string]any{},
				Display:       "Created " + namespace.Id,
				RemoteIP:      s.Location(),
				UserAgent:     s.UserAgent(),
				CorrelationID: "",
				Targets: []auditlog.AuditLogTarget{
					{
						Type:        auditlog.RatelimitNamespaceTargetType,
						ID:          namespace.Id,
						Name:        namespace.Name,
						DisplayName: namespace.Name,
						Meta:        nil,
					},
				},
			},
		})
	})
	if err != nil {
		return err
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: namespace,
	})
}
