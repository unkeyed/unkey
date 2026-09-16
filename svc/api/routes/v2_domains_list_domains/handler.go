package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/unkeyed/unkey/pkg/array"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/mysql"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/domain"
	"github.com/unkeyed/unkey/svc/api/internal/pagination"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2DomainsListDomainsRequestBody
	Response = openapi.V2DomainsListDomainsResponseBody
)

type Handler struct {
	DB db.Database
}

const scanLimit = 10_000

// errScanLimit distinguishes an incomplete authorized scan from a database failure.
var errScanLimit = errors.New("domain scan budget exhausted")

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/domains.listDomains"
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

	legacy := rbac.T(rbac.Tuple{ResourceType: rbac.Environment, ResourceID: "*", Action: rbac.ReadDomain})
	legacyAllowed := rbac.Check(legacy, principal.Permissions) == nil

	p := pagination.Parse(req.Limit, req.Cursor, 100)
	params, err := h.resolveDomainFilter(ctx, principal.AuthorizedWorkspaceID, req)
	if err != nil {
		return err
	}
	params.IDCursor = p.Cursor
	params.Search = mysql.SearchContains(strings.TrimSpace(ptr.SafeDeref(req.Search)))
	params.Limit = p.FetchLimit()
	rows, err := h.listAuthorized(ctx, principal, legacyAllowed, params)
	if errors.Is(err, errScanLimit) {
		s.SetInternalError(err.Error())
		return s.ProblemJSON(http.StatusServiceUnavailable, openapi.ServiceUnavailableErrorResponse{
			Meta: openapi.Meta{RequestId: s.RequestID()},
			Error: openapi.BaseError{
				Title:  "Service Unavailable",
				Type:   codes.App.Internal.ServiceUnavailable.DocsURL(),
				Detail: "The domain scan limit was reached. Narrow the project, app, environment, or search filters and retry.",
				Status: http.StatusServiceUnavailable,
			},
		})
	}
	if err != nil {
		return err
	}

	rows, pg := pagination.Paginate(rows, p, func(r db.ListCustomDomainsRow) string { return r.ID })

	data := array.Map(rows, func(row db.ListCustomDomainsRow) openapi.Domain {
		d := openapi.Domain{
			Id:                row.ID,
			Domain:            row.Domain,
			ProjectId:         row.ProjectID,
			AppId:             row.AppID,
			EnvironmentId:     row.EnvironmentID,
			Status:            domain.Status(row.VerificationStatus),
			VerificationError: nil,
			DnsRecords: domain.DnsRecords(domain.DnsRecordsInput{
				Domain:            row.Domain,
				TargetCname:       row.TargetCname,
				VerificationToken: row.VerificationToken,
				RoutingVerified:   row.CnameVerified,
				OwnershipVerified: row.OwnershipVerified,
			}),
			DomainConnect: nil,
			CreatedAt:     row.CreatedAt,
			UpdatedAt:     nil,
		}
		if row.DomainConnectProvider.Valid && row.DomainConnectUrl.Valid {
			d.DomainConnect = &openapi.DomainConnect{
				Provider: row.DomainConnectProvider.String,
				Url:      row.DomainConnectUrl.String,
			}
		}
		if row.VerificationError.Valid && row.VerificationError.String != "" {
			d.VerificationError = ptr.P(row.VerificationError.String)
		}
		if row.UpdatedAt.Valid {
			d.UpdatedAt = ptr.P(row.UpdatedAt.Int64)
		}
		return d
	})

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data:       data,
		Pagination: pg,
	})
}

// resolveDomainFilter runs only the most specific lookup. Broader filters constrain
// that lookup, so refills need only the resolved IDs, not repeated parent joins.
func (h *Handler) resolveDomainFilter(ctx context.Context, workspaceID string, req Request) (db.ListCustomDomainsParams, error) {
	var params db.ListCustomDomainsParams
	params.WorkspaceID = workspaceID
	params.Scope = ""
	var err error
	switch {
	case req.Environment != nil:
		params.Scope = "environment"
		params.EnvironmentIds, err = db.Query.ResolveCustomDomainEnvironments(ctx, h.DB.RO(), db.ResolveCustomDomainEnvironmentsParams{
			WorkspaceID: workspaceID,
			Project:     ptr.SafeDeref(req.Project),
			App:         ptr.SafeDeref(req.App),
			Environment: *req.Environment,
		})
	case req.App != nil:
		params.Scope = "app"
		params.AppIds, err = db.Query.ResolveCustomDomainApps(ctx, h.DB.RO(), db.ResolveCustomDomainAppsParams{
			WorkspaceID: workspaceID,
			Project:     ptr.SafeDeref(req.Project),
			App:         *req.App,
		})
	case req.Project != nil:
		params.Scope = "project"
		params.ProjectIds, err = db.Query.ResolveCustomDomainProjects(ctx, h.DB.RO(), db.ResolveCustomDomainProjectsParams{
			WorkspaceID: workspaceID,
			Project:     *req.Project,
		})
	}
	if err != nil {
		return db.ListCustomDomainsParams{}, fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("domain filter resolution failed"),
			fault.Public("Failed to retrieve domains."),
		)
	}
	return params, nil
}

// listAuthorized fills the page and its authorized lookahead without exposing denied row IDs.
// The raw lookahead remains inclusive so refills neither repeat nor skip candidates.
func (h *Handler) listAuthorized(ctx context.Context, subject *principal.Principal, legacyAllowed bool, params db.ListCustomDomainsParams) ([]db.ListCustomDomainsRow, error) {
	if params.Scope != "" && len(params.ProjectIds)+len(params.AppIds)+len(params.EnvironmentIds) == 0 {
		return nil, nil
	}
	wanted := int(params.Limit)
	rows := make([]db.ListCustomDomainsRow, 0, wanted)
	batchSize := wanted
	for scanned := 0; scanned < scanLimit; {
		batchSize = min(batchSize, scanLimit-scanned)
		params.Limit = int32(batchSize + 1) // nolint:gosec // bounded by scanLimit
		batch, err := db.Query.ListCustomDomains(ctx, h.DB.RO(), params)
		if err != nil {
			return nil, fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"),
				fault.Public("Failed to retrieve domains."),
			)
		}

		var nextBatchCursor string
		if len(batch) > batchSize {
			// The extra row proves another batch exists. Leave it unprocessed because
			// the next query includes the row at its cursor (id >= cursor).
			nextBatchCursor = batch[batchSize].ID
			batch = batch[:batchSize]
		}

		for _, row := range batch {
			scanned++
			if !legacyAllowed {
				query := rbac.U(
					urn.New().Workspace(subject.AuthorizedWorkspaceID).Project(row.ProjectID).App(row.AppID).Environment(row.EnvironmentID).Domain(row.ID),
					permissions.Read,
				)
				if rbac.Check(query, subject.Permissions) != nil {
					continue
				}
			}
			rows = append(rows, row)
			if len(rows) == wanted {
				return rows, nil
			}
		}
		if nextBatchCursor == "" {
			return rows, nil
		}
		params.IDCursor = nextBatchCursor
		// Grow refills to bound database calls for sparse grants; the remaining scan budget caps each batch.
		batchSize = max(batchSize*2, 100)
	}
	return nil, errScanLimit
}
