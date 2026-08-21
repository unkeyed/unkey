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
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/validation"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/portal"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2PortalUpdatePortalRequestBody
	Response = openapi.V2PortalUpdatePortalResponseBody
)

// notFoundMessage is the single public message every unresolved update returns.
//
// A denial, a portal in another workspace, and an unknown id or slug all share
// it, so no response body can be used to tell them apart.
const notFoundMessage = "Portal not found."

type Handler struct {
	DB        db.Database
	Auditlogs auditlogs.AuditLogService
	Clock     clock.Clock
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/portal.updatePortal"
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

	// Everything the request names is validated before the transaction opens.
	// Nothing here reads a row, so an invalid request never takes a write
	// connection on the primary.
	if req.Slug != nil && !validation.ValidateSlug(*req.Slug) {
		return fault.New("invalid portal slug",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal(fmt.Sprintf("slug %q failed validation", *req.Slug)),
			fault.Public(validation.ErrMsgInvalidSlug),
		)
	}

	// Tri-state: unspecified keeps the stored value, an explicit null clears the
	// column, and a value replaces it. Only the third case has anything to
	// validate.
	logoUrl := sql.NullString{String: "", Valid: false}
	if req.LogoUrl.IsSpecified() && !req.LogoUrl.IsNull() {
		value, getErr := req.LogoUrl.Get()
		if getErr != nil {
			return fault.Wrap(getErr,
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Internal("unable to read logoUrl"),
				fault.Public(portal.ErrMsgInvalidLogoURL),
			)
		}
		if err = portal.ValidateLogoURL(value); err != nil {
			return err
		}
		logoUrl = sql.NullString{String: value, Valid: true}
	}

	primaryColor := sql.NullString{String: "", Valid: false}
	if req.PrimaryColor.IsSpecified() && !req.PrimaryColor.IsNull() {
		value, getErr := req.PrimaryColor.Get()
		if getErr != nil {
			return fault.Wrap(getErr,
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Internal("unable to read primaryColor"),
				fault.Public(portal.ErrMsgInvalidColor),
			)
		}
		if err = portal.ValidatePrimaryColor(value); err != nil {
			return err
		}
		primaryColor = sql.NullString{String: value, Valid: true}
	}

	// Both association columns are derived from the one requested mapping and are
	// written as a pair, so "set the keyspace" on an app-mapped portal cannot
	// leave the app id behind. A row with both set, or neither, is unrepresentable
	// through this route.
	mappingAppID := sql.NullString{String: "", Valid: false}
	mappingKeyAuthID := sql.NullString{String: "", Valid: false}
	if req.Mapping != nil {
		mappingAppID, mappingKeyAuthID, err = portal.ColumnsFor(*req.Mapping)
		if err != nil {
			return err
		}
	}

	now := h.Clock.Now().UnixMilli()
	ctx = auditlog.WithCorrelation(ctx, auditlog.NewCorrelationID())

	// The resolve, the availability pre-check, the write, the session revocation,
	// and the audit entry have to be one atomic unit, so the closure is never
	// replayed.
	data, err := db.TxWithResult(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) (openapi.Portal, error) {
		var empty openapi.Portal

		found, err := db.Query.FindPortalByIdOrSlug(ctx, tx, db.FindPortalByIdOrSlugParams{
			Portal:      req.Portal,
			WorkspaceID: principal.WorkspaceID,
		})
		if err != nil {
			if db.IsNotFound(err) {
				return empty, fault.New("portal not found",
					fault.Code(codes.Data.Portal.NotFound.URN()),
					fault.Internal("no portal matched the request in this workspace"),
					fault.Public(notFoundMessage),
				)
			}
			return empty, fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error looking up portal"),
				fault.Public("We're unable to update the portal."),
			)
		}

		// Resolved first, then authorized, so the query can name the concrete id a
		// scoped grant would carry. Safe because the resolve is workspace-scoped --
		// a foreign portal is already absent above -- Authorize is an in-memory
		// check over already-loaded permissions, and nothing has been written yet.
		// The wildcard arm is spelled out separately because a stored `*` matches
		// literally and does not expand.
		//
		// The URN arms are what let the dashboard reach this route: its proxy mints
		// a token whose admin grant is a URN, so a legacy-only check would deny the
		// only operator surface there is.
		err = principal.Authorize(rbac.Or(
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Portal,
				ResourceID:   "*",
				Action:       rbac.UpdatePortal,
			}),
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Portal,
				ResourceID:   found.ID,
				Action:       rbac.UpdatePortal,
			}),
			rbac.U(
				urn.New().Workspace(principal.WorkspaceID).Portal("*"),
				permissions.UpdatePortal{},
			),
			rbac.U(
				urn.New().Workspace(principal.WorkspaceID).Portal(found.ID),
				permissions.UpdatePortal{},
			),
		))
		if err != nil {
			// A fresh chain, not a wrap: UserFacingMessage concatenates every public
			// message in the chain, so wrapping would append the rendered RBAC query
			// -- which names the resolved portal id -- to the response. A caller may
			// have addressed the portal by slug and never seen that id. The internal
			// message carries the denial across so logs still distinguish it from a
			// genuinely absent portal.
			return empty, fault.New("portal not found",
				fault.Code(codes.Data.Portal.NotFound.URN()),
				fault.Internal(fmt.Sprintf("update denied for portal %s: %s", found.ID, fault.InternalMessage(err))),
				fault.Public(notFoundMessage),
			)
		}

		if req.Mapping != nil {
			if err = portal.VerifyMappingOwned(ctx, tx, principal.WorkspaceID, *req.Mapping); err != nil {
				return empty, err
			}

			if err = portal.AuthorizeMappingTarget(ctx, tx, principal, principal.WorkspaceID, *req.Mapping); err != nil {
				return empty, err
			}
		}

		if err = h.assertAvailable(ctx, tx, principal.WorkspaceID, found, req); err != nil {
			return empty, err
		}

		params := db.UpdatePortalParams{
			WorkspaceID:           principal.WorkspaceID,
			ID:                    found.ID,
			UpdatedAt:             sql.NullInt64{Valid: true, Int64: now},
			SlugSpecified:         0,
			Slug:                  "",
			DisplayNameSpecified:  0,
			DisplayName:           "",
			AppIDSpecified:        0,
			AppID:                 sql.NullString{String: "", Valid: false},
			KeyAuthIDSpecified:    0,
			KeyAuthID:             sql.NullString{String: "", Valid: false},
			EnabledSpecified:      0,
			Enabled:               false,
			LogoUrlSpecified:      0,
			LogoUrl:               sql.NullString{String: "", Valid: false},
			PrimaryColorSpecified: 0,
			PrimaryColor:          sql.NullString{String: "", Valid: false},
		}

		// `after` is the row the response and the audit entry's after-state are both
		// composed from. It starts as the resolved row so an omitted field keeps its
		// stored value here exactly as the CASE expressions keep it in MySQL.
		after := found
		after.UpdatedAt = sql.NullInt64{Valid: true, Int64: now}

		if req.Slug != nil {
			params.Slug = *req.Slug
			params.SlugSpecified = 1
			after.Slug = *req.Slug
		}

		if req.DisplayName != nil {
			params.DisplayName = *req.DisplayName
			params.DisplayNameSpecified = 1
			after.DisplayName = *req.DisplayName
		}

		if req.Enabled != nil {
			params.Enabled = *req.Enabled
			params.EnabledSpecified = 1
			after.Enabled = *req.Enabled
		}

		// Both flags are set together or neither is. Setting one alone is the write
		// that could produce a row with both associations, which the application is
		// solely responsible for preventing.
		mappingChanged := false
		if req.Mapping != nil {
			params.AppID = mappingAppID
			params.AppIDSpecified = 1
			params.KeyAuthID = mappingKeyAuthID
			params.KeyAuthIDSpecified = 1
			after.AppID = mappingAppID
			after.KeyAuthID = mappingKeyAuthID
			// Compared through the same absent-semantics the rest of the package
			// uses, not raw NullString equality: a legacy row can hold a Valid but
			// empty column, and treating that as different from NULL would report a
			// change for an identical mapping and cut every live session.
			mappingChanged = !portal.SameAssociation(found.AppID, mappingAppID) ||
				!portal.SameAssociation(found.KeyAuthID, mappingKeyAuthID)
		}

		if req.LogoUrl.IsSpecified() {
			params.LogoUrl = logoUrl
			params.LogoUrlSpecified = 1
			after.LogoUrl = logoUrl
		}

		if req.PrimaryColor.IsSpecified() {
			params.PrimaryColor = primaryColor
			params.PrimaryColorSpecified = 1
			after.PrimaryColor = primaryColor
		}

		affected, err := db.Query.UpdatePortal(ctx, tx, params)
		if err != nil {
			// The pre-check above narrows the message for the collisions it can see,
			// but it is not a lock: `FindPortalByIdOrSlug` takes no row lock, so a
			// concurrent write can still win a unique key between the check and this
			// statement. This is the authoritative arm, kept generic because the
			// index name is not worth parsing out of the driver error and naming it
			// would leak which tenant holds the association.
			if db.IsDuplicateKeyError(err) {
				return empty, fault.Wrap(err,
					fault.Code(codes.Data.Portal.Duplicate.URN()),
					fault.Internal("duplicate key updating portal"),
					fault.Public("A portal already exists for that slug, app, or keyspace."),
				)
			}
			return empty, fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("unable to update portal"),
				fault.Public("We're unable to update the portal."),
			)
		}

		// MySQL reports zero affected rows both when the row is gone and when the
		// statement changed nothing, so re-read before calling it a delete: the
		// resolve takes no row lock, and a caller re-sending the values it already
		// stored writes an identical row.
		if affected == 0 {
			if _, err := db.Query.FindPortalByIdOrSlug(ctx, tx, db.FindPortalByIdOrSlugParams{
				Portal:      found.ID,
				WorkspaceID: principal.WorkspaceID,
			}); err != nil {
				return empty, fault.New("portal not found",
					fault.Code(codes.Data.Portal.NotFound.URN()),
					fault.Internal(fmt.Sprintf("update matched no rows for portal %s; concurrently deleted", found.ID)),
					fault.Public(notFoundMessage),
				)
			}
		}

		// A session freezes its keyspace scope at mint time and the session
		// resolver never reads `portals`, so re-pointing the mapping without this
		// would keep authenticating end users against the resource the portal no
		// longer serves. Re-sending the same mapping is not a change and must not
		// cut live sessions; neither does disabling the portal, which only stops new
		// sessions from being minted.
		var revoked int64
		if mappingChanged {
			revoked, err = db.Query.RevokePortalSessionsByPortal(ctx, tx, db.RevokePortalSessionsByPortalParams{
				RevokedAt:   sql.NullInt64{Valid: true, Int64: now},
				PortalID:    found.ID,
				WorkspaceID: principal.WorkspaceID,
			})
			if err != nil {
				return empty, fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("unable to revoke portal sessions"),
					fault.Public("We're unable to update the portal."),
				)
			}
		}

		beforeType, beforeID := portal.DescribeMapping(found)
		afterType, afterID := portal.DescribeMapping(after)

		err = h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
			{
				WorkspaceID:   principal.WorkspaceID,
				Event:         auditlog.PortalUpdateEvent,
				Display:       fmt.Sprintf("Updated portal %s", found.ID),
				ActorID:       principal.Subject.ID,
				ActorName:     principal.Subject.Name,
				ActorMeta:     map[string]any{},
				ActorType:     auditlog.AuditLogActor(principal.Subject.Type),
				RemoteIP:      s.Location(),
				UserAgent:     s.UserAgent(),
				CorrelationID: "",
				Resources: []auditlog.AuditLogResource{
					{
						ID:   found.ID,
						Type: auditlog.PortalResourceType,
						// Before and after both, because the mapping decides which
						// keyspaces every session minted from this portal can reach and
						// an incident reviewer cannot recover the previous value from the
						// row once it is overwritten.
						Meta: map[string]any{
							"slugBefore":        found.Slug,
							"slugAfter":         after.Slug,
							"mappingTypeBefore": beforeType,
							"mappingIdBefore":   beforeID,
							"mappingTypeAfter":  afterType,
							"mappingIdAfter":    afterID,
							"enabledBefore":     found.Enabled,
							"enabledAfter":      after.Enabled,
							"sessionsRevoked":   revoked,
						},
						Name:        after.Slug,
						DisplayName: after.Slug,
					},
				},
			},
		})
		if err != nil {
			return empty, err
		}

		// Composed in-process rather than re-selected: a post-commit read would go
		// to the read-only connection, where Vitess can serve a stale or missing
		// row.
		//
		// Tolerant of an already-broken row on purpose. A portal written before
		// these routes existed may hold both associations or neither, and refusing
		// to describe it would roll the whole transaction back -- so `enabled:
		// false` on a misconfigured portal would fail, leaving the operator no way
		// to switch it off. A request that names a mapping repairs the row on its
		// way through.
		return portal.ToResponseTolerant(after), nil
	})
	if err != nil {
		return err
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: data,
	})
}

// assertAvailable reports a conflict naming the input the caller should change.
//
// Without it every collision would surface through the driver's duplicate-key
// error, which cannot say whether the slug or the mapping was taken. Both checks
// exclude the portal being updated, so re-sending a portal's own slug or its own
// mapping is not a conflict. The mapping check is otherwise unscoped, because the
// app and keyspace unique keys span the whole table: a caller can collide with a
// portal in a workspace it cannot see, and being told to "pick another slug"
// would send it round a loop it can never win.
func (h *Handler) assertAvailable(
	ctx context.Context,
	tx db.DBTX,
	workspaceID string,
	found db.Portal,
	req Request,
) error {
	if req.Slug != nil && *req.Slug != found.Slug {
		sibling, err := db.Query.FindPortalByIdOrSlug(ctx, tx, db.FindPortalByIdOrSlugParams{
			Portal:      *req.Slug,
			WorkspaceID: workspaceID,
		})
		switch {
		case err == nil && sibling.ID != found.ID:
			return fault.New("portal slug taken",
				fault.Code(codes.Data.Portal.Duplicate.URN()),
				fault.Internal(fmt.Sprintf("slug %s already used in workspace %s", *req.Slug, workspaceID)),
				fault.Public("That slug is already in use. Choose a different slug."),
			)
		case err == nil, db.IsNotFound(err):
		default:
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error checking slug availability"),
				fault.Public("We're unable to update the portal."),
			)
		}
	}

	if req.Mapping == nil {
		return nil
	}

	var (
		claimant  string
		claimErr  error
		mappingID = req.Mapping.Id
	)
	switch req.Mapping.Type {
	case openapi.PortalMappingTypeApp:
		claimant, claimErr = db.Query.FindPortalIdByAppAnyWorkspace(ctx, tx,
			sql.NullString{String: mappingID, Valid: true})
	case openapi.PortalMappingTypeKeyspace:
		claimant, claimErr = db.Query.FindPortalIdByKeyspaceAnyWorkspace(ctx, tx,
			sql.NullString{String: mappingID, Valid: true})
	default:
		return portal.ErrUnknownMappingType(req.Mapping.Type)
	}

	switch {
	case claimErr == nil && claimant != found.ID:
		// Says that the resource is taken without saying by whom. The owning
		// workspace is never named, so a conflict cannot be used to probe another
		// tenant for which apps it has wired up.
		return fault.New("portal mapping taken",
			fault.Code(codes.Data.Portal.Duplicate.URN()),
			fault.Internal(fmt.Sprintf("mapping %s/%s already backs a portal", req.Mapping.Type, mappingID)),
			fault.Public("That app or keyspace already has a portal."),
		)
	case claimErr == nil, db.IsNotFound(claimErr):
		return nil
	default:
		return fault.Wrap(claimErr,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error checking mapping availability"),
			fault.Public("We're unable to update the portal."),
		)
	}
}
