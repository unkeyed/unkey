package portaldomain

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// DeletePortalDomains deletes every domain on a portal that is being deleted,
// each the way [Service.DeletePortalDomain] deletes one, in one transaction.
func (s *Service) DeletePortalDomains(
	ctx context.Context,
	req *connect.Request[ctrlv1.DeletePortalDomainsRequest],
) (*connect.Response[ctrlv1.DeletePortalDomainsResponse], error) {
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return nil, err
	}
	if err := assert.All(
		assert.NotEmpty(req.Msg.GetWorkspaceId(), "workspace_id is required"),
		assert.NotEmpty(req.Msg.GetPortalId(), "portal_id is required"),
	); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	rows, err := s.db.ListPortalDomainsByPortal(ctx, req.Msg.GetPortalId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	doms := make([]db.PortalDomain, 0, len(rows))
	for _, dom := range rows {
		if dom.WorkspaceID == req.Msg.GetWorkspaceId() {
			doms = append(doms, dom)
		}
	}

	for _, dom := range doms {
		s.cancelVerification(ctx, dom)
	}

	var deleted int32
	err = db.TxRetry(ctx, s.db.RW(), func(txCtx context.Context, tx db.DBTX) error {
		deleted = 0
		for _, dom := range doms {
			txErr := s.deleteDomain(txCtx, tx, req.Msg.GetActor(), dom)
			// A concurrent DeletePortalDomain removed it first.
			if connect.CodeOf(txErr) == connect.CodeNotFound {
				continue
			}
			if txErr != nil {
				return txErr
			}
			deleted++
		}
		return nil
	})
	if err != nil {
		if connectErr, ok := errors.AsType[*connect.Error](err); ok {
			return nil, connectErr
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&ctrlv1.DeletePortalDomainsResponse{Deleted: deleted}), nil
}
