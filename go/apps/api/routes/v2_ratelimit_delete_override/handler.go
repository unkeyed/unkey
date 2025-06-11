package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/internal/services/auditlogs"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/internal/services/permissions"
	"github.com/unkeyed/unkey/go/pkg/auditlog"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2RatelimitDeleteOverrideRequestBody
type Response = openapi.V2RatelimitDeleteOverrideResponseBody

// Handler implements zen.Route interface for the v2 ratelimit delete override endpoint
type Handler struct {
	// Services as public fields
	Logger      logging.Logger
	DB          db.Database
	Keys        keys.KeyService
	Permissions permissions.PermissionService
	Auditlogs   auditlogs.AuditLogService
}

// Method returns the HTTP method this route responds to
func (h *Handler) Method() string {
	return "POST"
}

// Path returns the URL path pattern this route matches
func (h *Handler) Path() string {
	return "/v2/ratelimit.deleteOverride"
}

// Handle processes the HTTP request
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	auth, err := h.Keys.VerifyRootKey(ctx, s)
	if err != nil {
		return err
	}

	// nolint:exhaustruct
	req := Request{}
	err = s.BindBody(&req)
	if err != nil {
		return fault.Wrap(err,
			fault.Internal("invalid request body"),
			fault.Public("The request body is invalid."),
		)
	}

	namespace, err := getNamespace(ctx, h, auth.AuthorizedWorkspaceID, req)
	if db.IsNotFound(err) {
		return fault.New("namespace not found",
			fault.Code(codes.Data.RatelimitNamespace.NotFound.URN()),
			fault.Internal("namespace not found"),
			fault.Public("This namespace does not exist."),
		)
	}
	if err != nil {
		return err
	}

	if namespace.WorkspaceID != auth.AuthorizedWorkspaceID {
		return fault.New("namespace not found",
			fault.Code(codes.Data.RatelimitNamespace.NotFound.URN()),
			fault.Internal("wrong workspace, masking as 404"),
			fault.Public("This namespace does not exist."),
		)
	}

	err = h.Permissions.Check(
		ctx,
		auth.KeyID,
		rbac.Or(
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Ratelimit,
				ResourceID:   namespace.ID,
				Action:       rbac.DeleteOverride,
			}),
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Ratelimit,
				ResourceID:   "*",
				Action:       rbac.DeleteOverride,
			}),
		),
	)
	if err != nil {
		return err
	}

	_, err = db.Tx(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) (interface{}, error) {
		// Check if the override exists before deleting
		override, err := db.Query.FindRatelimitOverrideByIdentifier(ctx, tx, db.FindRatelimitOverrideByIdentifierParams{
			WorkspaceID: auth.AuthorizedWorkspaceID,
			NamespaceID: namespace.ID,
			Identifier:  req.Identifier,
		})

		if db.IsNotFound(err) {
			return nil, fault.New("override not found",
				fault.Code(codes.Data.RatelimitOverride.NotFound.URN()),
				fault.Internal("override not found"),
				fault.Public("This override does not exist."),
			)
		}
		if err != nil {
			return nil, err
		}

		// Perform soft delete by updating the DeletedAt field
		err = db.Query.SoftDeleteRatelimitOverride(ctx, tx, db.SoftDeleteRatelimitOverrideParams{
			ID:  override.ID,
			Now: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})

		if err != nil {
			return nil, fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database failed to soft delete ratelimit override"),
				fault.Public("The database is unavailable."),
			)
		}

		err = h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
			{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Event:       auditlog.RatelimitDeleteOverrideEvent,
				Display:     fmt.Sprintf("Deleted override %s.", override.ID),
				ActorID:     auth.KeyID,
				Bucket:      auditlogs.DEFAULT_BUCKET,
				ActorType:   auditlog.RootKeyActor,
				ActorName:   "root key",
				RemoteIP:    s.Location(),
				UserAgent:   s.UserAgent(),
				Resources: []auditlog.AuditLogResource{
					{
						ID:          override.ID,
						Name:        override.Identifier,
						DisplayName: override.Identifier,
						Type:        auditlog.RatelimitOverrideResourceType,
						Meta:        nil,
					},
					{
						ID:          namespace.ID,
						Name:        namespace.Name,
						DisplayName: namespace.Name,
						Type:        auditlog.RatelimitNamespaceResourceType,
						Meta:        nil,
					},
				},
			},
		})
		if err != nil {
			return nil, fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database failed to insert audit logs"),
				fault.Public("Failed to insert audit logs"),
			)
		}

		return nil, nil
	})
	if err != nil {
		return err
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.RatelimitDeleteOverrideResponseData{},
	})
}

func getNamespace(ctx context.Context, h *Handler, workspaceID string, req Request) (db.RatelimitNamespace, error) {
	switch {
	case req.NamespaceId != nil:
		return db.Query.FindRatelimitNamespaceByID(ctx, h.DB.RO(), *req.NamespaceId)
	case req.NamespaceName != nil:
		return db.Query.FindRatelimitNamespaceByName(ctx, h.DB.RO(), db.FindRatelimitNamespaceByNameParams{
			WorkspaceID: workspaceID,
			Name:        *req.NamespaceName,
		})
	}

	return db.RatelimitNamespace{}, fault.New("missing namespace id or name",
		fault.Code(codes.App.Validation.InvalidInput.URN()),
		fault.Internal("missing namespace id or name"),
		fault.Public("You must provide either a namespace ID or name."),
	)
}
