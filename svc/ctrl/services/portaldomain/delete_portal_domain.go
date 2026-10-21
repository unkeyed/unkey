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
		q := db.NewQueries(tx)

		status, txErr := q.FindPortalDomainStatusByIdForUpdate(txCtx, dom.ID)
		if txErr != nil {
			if db.IsNotFound(txErr) {
				return connect.NewError(connect.CodeNotFound, fmt.Errorf("portal domain not found: %s", dom.ID))
			}
			return fmt.Errorf("failed to lock portal domain: %w", txErr)
		}

		// Every tenant's route shares the portal project, so the project scope
		// cannot tell tenants apart. Only the verified row for a hostname owns its
		// route; deleting an unverified claim must leave another tenant's in place.
		if status == db.PortalDomainsVerificationStatusVerified {
			env, envErr := q.FindEnvironmentById(txCtx, s.environmentID)
			if envErr != nil {
				return fmt.Errorf("failed to find portal environment: %w", envErr)
			}
			if txErr := q.DeleteFrontlineRouteByFQDNAndProject(txCtx, db.DeleteFrontlineRouteByFQDNAndProjectParams{
				Fqdn:      dom.Domain,
				ProjectID: env.ProjectID,
			}); txErr != nil && !db.IsNotFound(txErr) {
				return fmt.Errorf("failed to delete frontline route: %w", txErr)
			}
		}

		if txErr := q.DeleteAcmeChallengeByDomainID(txCtx, dom.ID); txErr != nil && !db.IsNotFound(txErr) {
			return fmt.Errorf("failed to delete ACME challenge: %w", txErr)
		}

		if txErr := q.DeletePortalDomainByID(txCtx, dom.ID); txErr != nil {
			return fmt.Errorf("failed to delete portal domain: %w", txErr)
		}

		return s.audit(txCtx, tx, req.Msg.GetActor(), dom.WorkspaceID, auditlog.PortalDomainDeleteEvent,
			fmt.Sprintf("Deleted portal domain %s", dom.Domain), dom.ID, dom.PortalID, dom.Domain)
	})
	if err != nil {
		if connectErr, ok := errors.AsType[*connect.Error](err); ok {
			return nil, connectErr
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&ctrlv1.DeletePortalDomainResponse{}), nil
}
