package portaldomain

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/domain/domaingate"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/gatefault"
)

// RetryPortalDomainVerification resets and restarts verification of a pending, verifying,
// or failed portal domain. A verified domain is rejected with FailedPrecondition.
func (s *Service) RetryPortalDomainVerification(
	ctx context.Context,
	req *connect.Request[ctrlv1.RetryPortalDomainVerificationRequest],
) (*connect.Response[ctrlv1.RetryPortalDomainVerificationResponse], error) {
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return nil, err
	}

	dom, err := s.findOwned(ctx, req.Msg.GetWorkspaceId(), req.Msg.GetPortalId(), req.Msg.GetDomainId())
	if err != nil {
		return nil, err
	}

	if dom.VerificationStatus == db.PortalDomainsVerificationStatusVerified {
		return nil, gatefault.ConnectWith(connect.CodeFailedPrecondition, domaingate.AlreadyVerified(dom.Domain))
	}

	s.cancelVerification(ctx, dom)

	now := time.Now().UnixMilli()

	err = db.TxRetry(ctx, s.db.RW(), func(txCtx context.Context, tx db.DBTX) error {
		if txErr := s.audit(txCtx, tx, req.Msg.GetActor(), dom.WorkspaceID, auditlog.PortalDomainVerifyEvent,
			fmt.Sprintf("Retried verification for portal domain %s", dom.Domain), dom.ID, dom.PortalID, dom.Domain); txErr != nil {
			return txErr
		}

		sendResp, sendErr := s.startVerification(txCtx, dom.ID)
		if sendErr != nil {
			return fmt.Errorf("failed to trigger verification: %w", sendErr)
		}

		if txErr := db.NewQueries(tx).ResetPortalDomainVerification(txCtx, db.ResetPortalDomainVerificationParams{
			ID:                 dom.ID,
			VerificationStatus: db.PortalDomainsVerificationStatusPending,
			CheckAttempts:      0,
			InvocationID:       sql.NullString{Valid: true, String: sendResp.Id()},
			UpdatedAt:          sql.NullInt64{Valid: true, Int64: now},
		}); txErr != nil {
			return fmt.Errorf("failed to reset verification: %w", txErr)
		}

		return nil
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&ctrlv1.RetryPortalDomainVerificationResponse{}), nil
}
