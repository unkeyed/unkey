// Package portaldomain implements the PortalDomainService ConnectRPC API, which
// adds, deletes, and retries verification of the hostnames tenants attach to
// their portal.
package portaldomain

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	restateingress "github.com/restatedev/sdk-go/ingress"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/logger"
	restateadmin "github.com/unkeyed/unkey/pkg/restate/admin"
	"github.com/unkeyed/unkey/svc/ctrl/internal/actor"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auditlogs"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// Service implements the PortalDomainService ConnectRPC API. It persists
// portal domains and delegates their verification to the Restate
// PortalDomainService virtual object.
type Service struct {
	ctrlv1connect.UnimplementedPortalDomainServiceHandler
	db            db.Database
	restate       *restateingress.Client
	restateAdmin  *restateadmin.Client
	auditlogs     auditlogs.AuditLogService
	cnameDomain   string
	environmentID string
	bearer        string
}

// Config holds the configuration for creating a new [Service].
type Config struct {
	Database db.Database
	// Restate is the ingress client that starts verification workflows.
	Restate *restateingress.Client
	// RestateAdmin cancels verification workflows. Nil skips cancellation.
	RestateAdmin *restateadmin.Client
	// Auditlogs records mutations within the same transaction as the write.
	Auditlogs auditlogs.AuditLogService
	// CnameDomain is the base domain for portal CNAME targets. Empty makes
	// AddPortalDomain fail.
	CnameDomain string
	// EnvironmentID is the portal app's production environment. Empty makes
	// AddPortalDomain fail, and it scopes the route a
	// verified domain's deletion removes.
	EnvironmentID string
	// Bearer is the preshared token callers must send in the Authorization header.
	Bearer string
}

// New creates a new [Service] with the given configuration.
func New(cfg Config) *Service {
	return &Service{
		UnimplementedPortalDomainServiceHandler: ctrlv1connect.UnimplementedPortalDomainServiceHandler{},
		db:                                      cfg.Database,
		restate:                                 cfg.Restate,
		restateAdmin:                            cfg.RestateAdmin,
		auditlogs:                               cfg.Auditlogs,
		cnameDomain:                             cfg.CnameDomain,
		environmentID:                           cfg.EnvironmentID,
		bearer:                                  cfg.Bearer,
	}
}

// findOwned loads a portal domain, answering NotFound when it belongs to
// another workspace or portal so the response cannot be used to probe ids.
func (s *Service) findOwned(ctx context.Context, workspaceID, portalID, domainID string) (db.PortalDomain, error) {
	dom, err := s.db.FindPortalDomainById(ctx, domainID)
	if err != nil {
		if db.IsNotFound(err) {
			return db.PortalDomain{}, connect.NewError(connect.CodeNotFound, fmt.Errorf("portal domain not found: %s", domainID))
		}
		return db.PortalDomain{}, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to find portal domain: %w", err))
	}
	if dom.WorkspaceID != workspaceID || dom.PortalID != portalID {
		return db.PortalDomain{}, connect.NewError(connect.CodeNotFound, fmt.Errorf("portal domain not found: %s", domainID))
	}
	return dom, nil
}

// cancelVerification cancels dom's running workflow, if any. A failed cancel
// is logged rather than returned: the workflow stops on its own once the row
// is gone or reset.
func (s *Service) cancelVerification(ctx context.Context, dom db.PortalDomain) {
	if !dom.InvocationID.Valid || s.restateAdmin == nil {
		return
	}
	if err := s.restateAdmin.CancelInvocation(ctx, dom.InvocationID.String); err != nil {
		logger.Warn("failed to cancel portal domain verification workflow",
			"domain", dom.Domain,
			"invocation_id", dom.InvocationID.String,
			"error", err,
		)
	}
}

// audit writes one portal domain audit entry inside tx.
func (s *Service) audit(
	ctx context.Context,
	tx db.DBTX,
	a *ctrlv1.ActorInfo,
	workspaceID string,
	event auditlog.AuditLogEvent,
	display, domainID, portalID, domain string,
) error {
	if err := s.auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
		{
			WorkspaceID:   workspaceID,
			Event:         event,
			Display:       display,
			ActorID:       a.GetId(),
			ActorName:     a.GetName(),
			ActorType:     actor.AuditType(a.GetType()),
			ActorMeta:     actor.Meta(a.GetMeta()),
			RemoteIP:      a.GetRemoteIp(),
			UserAgent:     a.GetUserAgent(),
			CorrelationID: "",
			Targets: []auditlog.AuditLogTarget{
				{
					ID:          domainID,
					Type:        auditlog.PortalDomainTargetType,
					Meta:        map[string]any{"domain": domain, "portalId": portalID},
					Name:        domain,
					DisplayName: domain,
				},
			},
		},
	}); err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}

// startVerification submits the verification workflow for domainID.
func (s *Service) startVerification(ctx context.Context, domainID string) (restateingress.SimpleSendResponse, error) {
	return hydrav1.NewPortalDomainServiceIngressClient(s.restate, domainID).
		VerifyDomain().
		Send(ctx, &hydrav1.VerifyPortalDomainRequest{})
}
