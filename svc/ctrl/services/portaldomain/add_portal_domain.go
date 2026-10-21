package portaldomain

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/domain/domaingate"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/gatefault"
)

// AddPortalDomain creates a portal domain and starts its verification workflow.
func (s *Service) AddPortalDomain(
	ctx context.Context,
	req *connect.Request[ctrlv1.AddPortalDomainRequest],
) (*connect.Response[ctrlv1.AddPortalDomainResponse], error) {
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return nil, err
	}
	if err := assert.All(
		assert.NotEmpty(req.Msg.GetWorkspaceId(), "workspace_id is required"),
		assert.NotEmpty(req.Msg.GetPortalId(), "portal_id is required"),
		assert.NotEmpty(req.Msg.GetDomain(), "domain is required"),
	); err != nil {
		// Not InvalidArgument, which is reserved for gatefault codes the API reflects.
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if s.environmentID == "" || s.cnameDomain == "" {
		return nil, gatefault.Connect(portalDomainsNotConfigured())
	}

	domain, parseErr := domaingate.ParseDomain(req.Msg.GetDomain())
	if parseErr != nil {
		return nil, gatefault.ConnectWith(connect.CodeInvalidArgument, parseErr)
	}

	workspaceID := req.Msg.GetWorkspaceId()

	_, err := s.db.FindPortalDomainByWorkspaceAndDomain(ctx, db.FindPortalDomainByWorkspaceAndDomainParams{
		WorkspaceID: workspaceID,
		Domain:      domain,
	})
	if err == nil {
		return nil, gatefault.ConnectWith(connect.CodeAlreadyExists, domaingate.AlreadyExists(domain))
	}
	if !db.IsNotFound(err) {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to check existing portal domain: %w", err))
	}

	// Concurrent creates can both pass, overshooting by at most the request
	// concurrency, which beats serializing every create in a workspace.
	attached, err := s.db.CountPortalDomainsByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to count portal domains: %w", err))
	}
	if attached >= maxPortalDomainsPerWorkspace {
		return nil, gatefault.ConnectWith(connect.CodeResourceExhausted, portalDomainCapReached(attached))
	}

	targetCname := fmt.Sprintf("%s.%s", uid.DNS1035(16), s.cnameDomain)
	verificationToken := uid.Secure(24)
	domainID := uid.New(uid.PortalDomainPrefix)
	now := time.Now().UnixMilli()

	err = db.TxRetry(ctx, s.db.RW(), func(txCtx context.Context, tx db.DBTX) error {
		q := db.NewQueries(tx)

		// The API authorized the portal against a replica, so re-check it on the
		// primary, and hold the lock so a concurrent portal delete waits.
		_, txErr := q.LockPortal(txCtx, db.LockPortalParams{
			ID:          req.Msg.GetPortalId(),
			WorkspaceID: workspaceID,
		})
		if txErr != nil {
			if db.IsNotFound(txErr) {
				return connect.NewError(connect.CodeNotFound, fmt.Errorf("portal not found: %s", req.Msg.GetPortalId()))
			}
			return connect.NewError(connect.CodeInternal, fmt.Errorf("lock portal: %w", txErr))
		}

		// One hostname routes one way, so a workspace cannot hold it as both a
		// deploy domain and a portal domain. No unique index spans the two tables.
		_, txErr = q.FindCustomDomainIDByWorkspaceAndDomain(txCtx, db.FindCustomDomainIDByWorkspaceAndDomainParams{
			WorkspaceID: workspaceID,
			Domain:      domain,
		})
		if txErr == nil {
			return gatefault.ConnectWith(connect.CodeAlreadyExists, domaingate.AlreadyExists(domain))
		}
		if !db.IsNotFound(txErr) {
			return connect.NewError(connect.CodeInternal, fmt.Errorf("check custom domain: %w", txErr))
		}

		if txErr := q.InsertPortalDomain(txCtx, db.InsertPortalDomainParams{
			ID:                    domainID,
			WorkspaceID:           workspaceID,
			PortalID:              req.Msg.GetPortalId(),
			Domain:                domain,
			VerificationStatus:    db.PortalDomainsVerificationStatusPending,
			VerificationToken:     verificationToken,
			TargetCname:           targetCname,
			DomainConnectProvider: sql.NullString{Valid: false, String: ""},
			DomainConnectUrl:      sql.NullString{Valid: false, String: ""},
			InvocationID:          sql.NullString{Valid: false, String: ""},
			CreatedAt:             now,
		}); txErr != nil {
			if db.IsDuplicateKeyError(txErr) {
				return gatefault.ConnectWith(connect.CodeAlreadyExists, domaingate.AlreadyExists(domain))
			}
			return connect.NewError(connect.CodeInternal, fmt.Errorf("insert portal domain: %w", txErr))
		}

		if txErr := s.audit(txCtx, tx, req.Msg.GetActor(), workspaceID, auditlog.PortalDomainCreateEvent,
			fmt.Sprintf("Added portal domain %s", domain), domainID, req.Msg.GetPortalId(), domain); txErr != nil {
			return connect.NewError(connect.CodeInternal, txErr)
		}

		// Submitting inside the transaction makes the RPC all-or-nothing: a failed
		// submit rolls back the row and its audit entry. The workflow tolerates
		// reading before the commit lands, and a TxRetry rerun re-submitting is
		// safe because the virtual object is keyed by domainID.
		sendResp, sendErr := s.startVerification(txCtx, domainID)
		if sendErr != nil {
			logger.Error("failed to trigger portal domain verification",
				"domain", domain,
				"domain_id", domainID,
				"error", sendErr,
			)
			return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to trigger verification workflow: %w", sendErr))
		}

		if txErr := q.UpdatePortalDomainInvocationID(txCtx, db.UpdatePortalDomainInvocationIDParams{
			ID:           domainID,
			InvocationID: sql.NullString{Valid: true, String: sendResp.Id()},
			UpdatedAt:    sql.NullInt64{Valid: true, Int64: now},
		}); txErr != nil {
			return connect.NewError(connect.CodeInternal, fmt.Errorf("record invocation id: %w", txErr))
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&ctrlv1.AddPortalDomainResponse{
		DomainId:          domainID,
		TargetCname:       targetCname,
		VerificationToken: verificationToken,
	}), nil
}
