package handler

import (
	"context"
	"database/sql"
	"net/http"
	"sort"

	"github.com/unkeyed/unkey/pkg/array"
	"github.com/unkeyed/unkey/pkg/auth/portalrbac"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/pagination"
	"github.com/unkeyed/unkey/svc/api/internal/portalscope"
	"github.com/unkeyed/unkey/svc/api/openapi"
	listkeys "github.com/unkeyed/unkey/svc/api/routes/v2_apis_list_keys"
)

type (
	// Request is the portal.listKeys public contract. Unlike apis.listKeys it has
	// no externalId or apiId: the listing is always scoped to the session's own
	// end user, within the keyspaces configured on the portal configuration.
	Request  = openapi.V2PortalListKeysRequestBody
	Response = openapi.V2PortalListKeysResponseBody
)

// Handler serves the portal-scoped key listing. It authenticates only portal
// sessions and lists the session end user's keys across the keyspaces the portal
// configuration is scoped to. It does not reuse the apis.listKeys core because
// that core is keyed by a caller-supplied apiId; here the keyspaces come from the
// session, so it queries them directly and reuses only the shared response shape.
type Handler struct {
	DB db.Database
}

// New builds a portal.listKeys handler.
func New(database db.Database) *Handler {
	return &Handler{DB: database}
}

// Method returns the HTTP method this route responds to.
func (h *Handler) Method() string { return "POST" }

// Path returns the URL path pattern this route matches.
func (h *Handler) Path() string { return "/v2/portal.listKeys" }

// Handle lists the portal session end user's keys, scoped to the session's
// configured keyspaces.
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	externalID, err := portalscope.ExternalID(s)
	if err != nil {
		return err
	}

	keyspaceIDs, err := portalscope.KeyspaceIDs(s)
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	p := pagination.Parse(req.Limit, req.Cursor, 100)

	if err := principal.Authorize(rbac.S(portalrbac.CapKeysRead)); err != nil {
		return err
	}

	// A session with no keyspaces can never see any keys.
	if len(keyspaceIDs) == 0 {
		return h.emptyResponse(s)
	}

	// Scope to the end user's own keys. If the identity does not exist yet, the
	// user simply has no keys.
	identity, err := db.Query.FindIdentityByExternalID(ctx, h.DB.RO(), db.FindIdentityByExternalIDParams{
		WorkspaceID: principal.WorkspaceID,
		ExternalID:  externalID,
		Deleted:     false,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return h.emptyResponse(s)
		}
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve identity."),
		)
	}

	// Fetch limit+1 across all scoped keyspaces so the DB does the id-ordered
	// pagination; a single keyspace is the common case.
	keyResults, err := db.Query.ListLiveKeysByKeySpaceIDs(ctx, h.DB.RO(), db.ListLiveKeysByKeySpaceIDsParams{
		KeySpaceIds: keyspaceIDs,
		IDCursor:    p.Cursor,
		IdentityID:  sql.NullString{String: identity.ID, Valid: true},
		Limit:       p.FetchLimit(),
	})
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve keys."),
		)
	}

	// Keep pagination deterministic across keyspaces (the query already orders by
	// key id, so this is defensive and free for the common single-keyspace case).
	sort.Slice(keyResults, func(i, j int) bool { return keyResults[i].ID < keyResults[j].ID })

	keyResults, pg := pagination.Paginate(keyResults, p, func(r db.ListLiveKeysByKeySpaceIDsRow) string { return r.ID })

	// Portal sessions never decrypt, so no plaintext is ever included.
	responseData := array.Map(keyResults, func(key db.ListLiveKeysByKeySpaceIDsRow) openapi.KeyResponseData {
		return listkeys.BuildKeyResponseData(db.ToKeyData(key), "")
	})

	return s.JSON(http.StatusOK, Response{
		Meta:       openapi.Meta{RequestId: s.RequestID()},
		Data:       responseData,
		Pagination: pg,
	})
}

// emptyResponse returns a well-formed empty page.
func (h *Handler) emptyResponse(s *zen.Session) error {
	return s.JSON(http.StatusOK, Response{
		Meta:       openapi.Meta{RequestId: s.RequestID()},
		Data:       []openapi.KeyResponseData{},
		Pagination: openapi.Pagination{Cursor: nil, HasMore: false},
	})
}
