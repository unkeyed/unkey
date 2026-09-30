package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	portalservice "github.com/unkeyed/unkey/internal/services/portal"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/mysql"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/pagination"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2PortalListSessionsRequestBody
	Response = openapi.V2PortalListSessionsResponseBody
)

// notFoundMessage is shared by an unknown portal and a denied caller, so the two
// look the same.
const notFoundMessage = "Portal not found."

type Handler struct {
	DB    db.Database
	Clock clock.Clock
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/portal.listSessions"
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

	found, err := db.Query.FindPortalByIdOrSlug(ctx, h.DB.RO(), db.FindPortalByIdOrSlugParams{
		Portal:      req.Portal,
		WorkspaceID: principal.AuthorizedWorkspaceID,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New("portal not found",
				fault.Code(codes.Data.Portal.NotFound.URN()),
				fault.Internal("no portal matched the request in this workspace"),
				fault.Public(notFoundMessage),
			)
		}
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error looking up portal"),
			fault.Public("We're unable to list the portal sessions."),
		)
	}

	err = principal.Authorize(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Portal,
			ResourceID:   "*",
			Action:       rbac.ReadPortal,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Portal,
			ResourceID:   found.ID,
			Action:       rbac.ReadPortal,
		}),
		rbac.U(
			urn.New().Workspace(principal.AuthorizedWorkspaceID).Project(found.ProjectID).Portal(found.ID),
			permissions.Read,
		),
	))
	if err != nil {
		// A fresh chain, not a wrap, so the rendered RBAC query and its portal id
		// stay out of the public message.
		return fault.New("portal not found",
			fault.Code(codes.Data.Portal.NotFound.URN()),
			fault.Internal(fmt.Sprintf("read denied for portal %s: %s", found.ID, fault.InternalMessage(err))),
			fault.Public(notFoundMessage),
		)
	}

	now := h.Clock.Now().UnixMilli()
	p := pagination.Parse(req.Limit, req.Cursor, 100)

	externalIDs, err := db.Query.ListLivePortalSessionExternalIDs(ctx, h.DB.RO(), db.ListLivePortalSessionExternalIDsParams{
		WorkspaceID:              principal.AuthorizedWorkspaceID,
		PortalID:                 found.ID,
		AccessTokenExpiresAfter:  sql.NullInt64{Valid: true, Int64: now},
		ExchangeCodeExpiresAfter: now,
		ExternalIDCursor:         p.Cursor,
		Search:                   mysql.SearchPrefix(ptr.SafeDeref(req.Search)),
		Limit:                    p.FetchLimit(),
	})
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal(fmt.Sprintf("unable to list session end users for portal %s", found.ID)),
			fault.Public("We're unable to list the portal sessions."),
		)
	}
	externalIDs, pg := pagination.Paginate(externalIDs, p, func(id string) string { return id })

	groups := []openapi.V2PortalListSessionsResponseData{}
	if len(externalIDs) > 0 {
		rows, err := db.Query.ListLivePortalSessionsByExternalIDs(ctx, h.DB.RO(), db.ListLivePortalSessionsByExternalIDsParams{
			WorkspaceID:              principal.AuthorizedWorkspaceID,
			PortalID:                 found.ID,
			ExternalIds:              externalIDs,
			AccessTokenExpiresAfter:  sql.NullInt64{Valid: true, Int64: now},
			ExchangeCodeExpiresAfter: now,
		})
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal(fmt.Sprintf("unable to list sessions for portal %s", found.ID)),
				fault.Public("We're unable to list the portal sessions."),
			)
		}

		groups, err = groupSessions(externalIDs, rows)
		if err != nil {
			return err
		}
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data:       groups,
		Pagination: pg,
	})
}

// groupSessions arranges rows under their end user in the order of externalIDs.
// An end user whose sessions all lapsed between the two reads has no rows and is
// dropped, so no group is ever empty.
func groupSessions(externalIDs []string, rows []db.ListLivePortalSessionsByExternalIDsRow) ([]openapi.V2PortalListSessionsResponseData, error) {
	byExternalID := make(map[string][]openapi.V2PortalListSessionsSession, len(externalIDs))
	for _, row := range rows {
		session, err := toSession(row)
		if err != nil {
			return nil, err
		}
		byExternalID[row.ExternalID] = append(byExternalID[row.ExternalID], session)
	}

	groups := make([]openapi.V2PortalListSessionsResponseData, 0, len(externalIDs))
	for _, externalID := range externalIDs {
		sessions, ok := byExternalID[externalID]
		if !ok {
			continue
		}
		groups = append(groups, openapi.V2PortalListSessionsResponseData{
			ExternalId: externalID,
			Sessions:   sessions,
		})
	}
	return groups, nil
}

// toSession maps a row to its response shape. A session without an access token
// was never exchanged, so it is pending and expires with its exchange code.
func toSession(row db.ListLivePortalSessionsByExternalIDsRow) (openapi.V2PortalListSessionsSession, error) {
	var grant portalservice.Grant
	if err := json.Unmarshal(row.Scopes, &grant); err != nil {
		return openapi.V2PortalListSessionsSession{}, fault.Wrap(err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal(fmt.Sprintf("malformed scopes on portal session %s", row.ID)),
			fault.Public("We're unable to list the portal sessions."),
		)
	}

	status := openapi.Active
	expiresAt := row.AccessTokenExpiresAt.Int64
	if !row.AccessTokenHash.Valid {
		status = openapi.Pending
		expiresAt = row.ExchangeCodeExpiresAt
	}

	scopes := grant.Scopes
	if scopes == nil {
		scopes = []string{}
	}

	return openapi.V2PortalListSessionsSession{
		Id:        row.ID,
		Status:    status,
		CreatedAt: row.CreatedAt,
		ExpiresAt: expiresAt,
		Scopes:    scopes,
	}, nil
}
