package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/unkeyed/unkey/pkg/array"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/mysql"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/domain"
	apierrors "github.com/unkeyed/unkey/svc/api/internal/errors"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2DomainsListDomainsRequestBody
	Response = openapi.V2DomainsListDomainsResponseBody
)

const maxDomains = 10

type Handler struct {
	DB db.Database
}

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

	env, err := db.Query.FindEnvironmentByIdentifiers(ctx, h.DB.RO(), db.FindEnvironmentByIdentifiersParams{
		WorkspaceID: principal.WorkspaceID,
		Project:     req.Project,
		App:         req.App,
		Environment: req.Environment,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New(
				"environment not found",
				fault.Code(codes.Data.Environment.NotFound.URN()),
				fault.Internal("environment not found"),
				fault.Public("The requested environment does not exist."),
			)
		}
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve environment."),
		)
	}

	if err = principal.Authorize(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Environment,
			ResourceID:   "*",
			Action:       rbac.ReadDomain,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Environment,
			ResourceID:   env.ID,
			Action:       rbac.ReadDomain,
		}),
	)); err != nil {
		return apierrors.MaskInsufficientPermissionsAsNotFound(
			err,
			codes.Data.Environment.NotFound.URN(),
			"The requested environment does not exist.",
		)
	}

	search := mysql.SearchContains(strings.TrimSpace(ptr.SafeDeref(req.Search)))

	rows, err := db.Query.ListCustomDomainsByEnvironment(ctx, h.DB.RO(), db.ListCustomDomainsByEnvironmentParams{
		ProjectID:     env.ProjectID,
		EnvironmentID: env.ID,
		Search:        search,
		Limit:         maxDomains,
	})
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve domains."),
		)
	}

	data := array.Map(rows, func(row db.ListCustomDomainsByEnvironmentRow) openapi.Domain {
		d := openapi.Domain{
			Id:            row.ID,
			Domain:        row.Domain,
			ProjectId:     row.ProjectID,
			AppId:         row.AppID,
			EnvironmentId: row.EnvironmentID,
			Status:        domain.Status(row.VerificationStatus),
			// The stored flag is named for the CNAME, but an apex domain routes through
			// an alias, so the API reports it by purpose instead of by record type.
			RoutingVerified:   row.CnameVerified,
			OwnershipVerified: row.OwnershipVerified,
			VerificationError: nil,
			LastCheckedAt:     nil,
			DnsRecords:        domain.DnsRecords(row.Domain, row.TargetCname, row.VerificationToken),
			CreatedAt:         row.CreatedAt,
			UpdatedAt:         nil,
		}
		if row.VerificationError.Valid && row.VerificationError.String != "" {
			d.VerificationError = ptr.P(row.VerificationError.String)
		}
		if row.LastCheckedAt.Valid {
			d.LastCheckedAt = ptr.P(row.LastCheckedAt.Int64)
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
		Data: data,
	})
}
