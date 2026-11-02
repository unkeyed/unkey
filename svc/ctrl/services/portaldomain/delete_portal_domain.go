package portaldomain

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// DeletePortalDomain stops verification and deletes the domain, its ACME
// challenge, and, when it is the verified row, its route. The certificate stays.
func (s *Service) DeletePortalDomain(
	ctx context.Context,
	req *connect.Request[ctrlv1.DeletePortalDomainRequest],
) (*connect.Response[ctrlv1.DeletePortalDomainResponse], error) {
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return nil, err
	}

	dom, err := s.findOwned(ctx, req.Msg.GetWorkspaceId(), req.Msg.GetPortalId(), req.Msg.GetDomainId())
	if err != nil {
		return nil, err
	}

	s.cancelVerification(ctx, dom)

	err = db.TxRetry(ctx, s.db.RW(), func(txCtx context.Context, tx db.DBTX) error {
		return s.deleteDomain(txCtx, tx, req.Msg.GetActor(), dom)
	})
	if err != nil {
		if connectErr, ok := errors.AsType[*connect.Error](err); ok {
			return nil, connectErr
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&ctrlv1.DeletePortalDomainResponse{}), nil
}

// deleteDomain removes dom, its ACME challenge, and, when it is the verified
// row, its route, and audits the delete, all inside tx.
func (s *Service) deleteDomain(ctx context.Context, tx db.DBTX, actor *ctrlv1.ActorInfo, dom db.PortalDomain) error {
	q := db.NewQueries(tx)
	display := fmt.Sprintf("Deleted portal domain %s and its ACME challenge", dom.Domain)

	status, err := q.FindPortalDomainStatusByIdForUpdate(ctx, dom.ID)
	if err != nil {
		if db.IsNotFound(err) {
			return connect.NewError(connect.CodeNotFound, fmt.Errorf("portal domain not found: %s", dom.ID))
		}
		return fmt.Errorf("failed to lock portal domain: %w", err)
	}

	// Every tenant's route shares the portal project, so the project scope
	// cannot tell tenants apart. Only the verified row for a hostname owns its
	// route; deleting an unverified claim must leave another tenant's in place.
	if status == db.PortalDomainsVerificationStatusVerified {
		env, envErr := q.FindEnvironmentById(ctx, s.environmentID)
		if envErr != nil {
			return fmt.Errorf("failed to find portal environment: %w", envErr)
		}
		if err := q.DeleteFrontlineRouteByFQDNAndProject(ctx, db.DeleteFrontlineRouteByFQDNAndProjectParams{
			Fqdn:      dom.Domain,
			ProjectID: env.ProjectID,
		}); err != nil && !db.IsNotFound(err) {
			return fmt.Errorf("failed to delete frontline route: %w", err)
		}
		display = fmt.Sprintf("Deleted portal domain %s, its ACME challenge, and its route", dom.Domain)
	}

	if err := q.DeleteAcmeChallengeByDomainID(ctx, dom.ID); err != nil && !db.IsNotFound(err) {
		return fmt.Errorf("failed to delete ACME challenge: %w", err)
	}

	if err := q.DeletePortalDomainByID(ctx, dom.ID); err != nil {
		return fmt.Errorf("failed to delete portal domain: %w", err)
	}

	return s.audit(ctx, tx, actor, dom.WorkspaceID, auditlog.PortalDomainDeleteEvent, display, dom.ID, dom.PortalID, dom.Domain)
}
