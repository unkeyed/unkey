package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/validation"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/portal"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2PortalCreatePortalRequestBody
	Response = openapi.V2PortalCreatePortalResponseBody
)

type Handler struct {
	DB        db.Database
	Auditlogs auditlogs.AuditLogService
	Clock     clock.Clock
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/portal.createPortal"
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

	if !validation.ValidateSlug(req.Slug) {
		return fault.New("invalid portal slug",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal(fmt.Sprintf("slug %q failed validation", req.Slug)),
			fault.Public(validation.ErrMsgInvalidSlug),
		)
	}

	logoUrl := sql.NullString{String: "", Valid: false}
	if req.LogoUrl != nil {
		if err = portal.ValidateLogoURL(*req.LogoUrl); err != nil {
			return err
		}
		logoUrl = sql.NullString{String: *req.LogoUrl, Valid: true}
	}

	primaryColor := sql.NullString{String: "", Valid: false}
	if req.PrimaryColor != nil {
		if err = portal.ValidatePrimaryColor(*req.PrimaryColor); err != nil {
			return err
		}
		primaryColor = sql.NullString{String: *req.PrimaryColor, Valid: true}
	}

	// Defaults to enabled: a portal nobody can mint a session for is the unusual
	// case, so it has to be asked for.
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	appID, keyAuthID, err := portal.ColumnsFor(req.Mapping)
	if err != nil {
		return err
	}

	// Only a wildcard grant can authorize a create: the portal id is minted below,
	// so no grant can name it yet. The URN arm is what lets the dashboard reach
	// this route, because its proxy mints a token whose admin grant is a URN.
	err = principal.Authorize(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Portal,
			ResourceID:   "*",
			Action:       rbac.CreatePortal,
		}),
		rbac.U(
			urn.New().Workspace(principal.WorkspaceID).Portal("*"),
			permissions.CreatePortal{},
		),
	))
	if err != nil {
		// Returned as-is rather than masked as a 404: there is no portal yet whose
		// existence a denial could disclose.
		return err
	}

	portalID := uid.New(uid.PortalPrefix)
	now := h.Clock.Now().UnixMilli()
	ctx = auditlog.WithCorrelation(ctx, auditlog.NewCorrelationID())

	err = db.Tx(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		if err := portal.VerifyMappingOwned(ctx, tx, principal.WorkspaceID, req.Mapping); err != nil {
			return err
		}

		if err := portal.AuthorizeMappingTarget(ctx, tx, principal, principal.WorkspaceID, req.Mapping); err != nil {
			return err
		}

		if err := h.assertAvailable(ctx, tx, principal.WorkspaceID, req.Slug, req.Mapping); err != nil {
			return err
		}

		err := db.Query.InsertPortal(ctx, tx, db.InsertPortalParams{
			ID:           portalID,
			WorkspaceID:  principal.WorkspaceID,
			Slug:         req.Slug,
			DisplayName:  req.DisplayName,
			AppID:        appID,
			KeyAuthID:    keyAuthID,
			Enabled:      enabled,
			LogoUrl:      logoUrl,
			PrimaryColor: primaryColor,
			CreatedAt:    now,
			UpdatedAt:    sql.NullInt64{Valid: false, Int64: 0},
		})
		if err != nil {
			// The pre-check above narrows the message for the cases it can see, but
			// it is not a lock: a concurrent create can still win the unique key
			// between the check and the insert. This is the authoritative arm, kept
			// generic because the index name is not worth parsing out of the driver
			// error and naming it would leak which tenant holds the association.
			if db.IsDuplicateKeyError(err) {
				return fault.Wrap(err,
					fault.Code(codes.Data.Portal.Duplicate.URN()),
					fault.Internal("duplicate key inserting portal"),
					fault.Public("A portal already exists for that slug, app, or keyspace."),
				)
			}
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("unable to insert portal"),
				fault.Public("We're unable to create the portal."),
			)
		}

		return h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
			{
				WorkspaceID:   principal.WorkspaceID,
				Event:         auditlog.PortalCreateEvent,
				Display:       fmt.Sprintf("Created portal %s", portalID),
				ActorID:       principal.Subject.ID,
				ActorName:     principal.Subject.Name,
				ActorMeta:     map[string]any{},
				ActorType:     auditlog.AuditLogActor(principal.Subject.Type),
				RemoteIP:      s.Location(),
				UserAgent:     s.UserAgent(),
				CorrelationID: "",
				Resources: []auditlog.AuditLogResource{
					{
						ID:   portalID,
						Type: auditlog.PortalResourceType,
						// The mapping decides which keyspaces every session minted
						// from this portal can reach, so an incident reviewer needs it
						// recorded rather than inferred from the row's later state.
						Meta: map[string]any{
							"slug":        req.Slug,
							"displayName": req.DisplayName,
							"mappingType": string(req.Mapping.Type),
							"mappingId":   req.Mapping.Id,
							"enabled":     enabled,
						},
						Name:        req.Slug,
						DisplayName: req.Slug,
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
		Data: openapi.V2PortalCreatePortalResponseData{
			PortalId: portalID,
		},
	})
}

// assertAvailable reports a conflict naming the input the caller should change.
//
// Without it every collision would surface through the driver's duplicate-key
// error, which cannot say whether the slug or the mapping was taken. The mapping
// check is deliberately unscoped, because the app and keyspace unique keys span
// the whole table: a caller can collide with a portal in a workspace it cannot
// see, and being told to "pick another slug" would send it round a loop it can
// never win.
func (h *Handler) assertAvailable(
	ctx context.Context,
	tx db.DBTX,
	workspaceID string,
	slug string,
	mapping openapi.PortalMapping,
) error {
	_, err := db.Query.FindPortalByIdOrSlug(ctx, tx, db.FindPortalByIdOrSlugParams{
		Portal:      slug,
		WorkspaceID: workspaceID,
	})
	switch {
	case err == nil:
		return fault.New("portal slug taken",
			fault.Code(codes.Data.Portal.Duplicate.URN()),
			fault.Internal(fmt.Sprintf("slug %s already used in workspace %s", slug, workspaceID)),
			fault.Public("That slug is already in use. Choose a different slug."),
		)
	case db.IsNotFound(err):
	default:
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error checking slug availability"),
			fault.Public("We're unable to create the portal."),
		)
	}

	var claimErr error
	switch mapping.Type {
	case openapi.PortalMappingTypeApp:
		_, claimErr = db.Query.FindPortalIdByAppAnyWorkspace(ctx, tx,
			sql.NullString{String: mapping.Id, Valid: true})
	case openapi.PortalMappingTypeKeyspace:
		_, claimErr = db.Query.FindPortalIdByKeyspaceAnyWorkspace(ctx, tx,
			sql.NullString{String: mapping.Id, Valid: true})
	default:
		return portal.ErrUnknownMappingType(mapping.Type)
	}

	switch {
	case claimErr == nil:
		// Says that the resource is taken without saying by whom. The owning
		// workspace is never named, so a conflict cannot be used to probe another
		// tenant for which apps it has wired up.
		return fault.New("portal mapping taken",
			fault.Code(codes.Data.Portal.Duplicate.URN()),
			fault.Internal(fmt.Sprintf("mapping %s/%s already backs a portal", mapping.Type, mapping.Id)),
			fault.Public("That app or keyspace already has a portal."),
		)
	case db.IsNotFound(claimErr):
		return nil
	default:
		return fault.Wrap(claimErr,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error checking mapping availability"),
			fault.Public("We're unable to create the portal."),
		)
	}
}
