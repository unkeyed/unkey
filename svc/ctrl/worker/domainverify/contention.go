package domainverify

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// ClaimSource names the table a hostname claim lives in. The values are the
// literals FindVerifiedDomainClaimExcludingWorkspace selects.
type ClaimSource string

const (
	ClaimSourceCustom ClaimSource = "custom"
	ClaimSourcePortal ClaimSource = "portal"
)

// revokedReason is what the losing workspace reads on its row.
const revokedReason = "domain claimed by another workspace"

// RevokeContestedClaim revokes another workspace's verified claim on hostname,
// if one exists: the row is marked failed, the hostname's route is deleted and
// its pending ACME challenge removed. workspaceID is the workspace that just
// proved ownership. Every write is idempotent, so the step can be retried.
func RevokeContestedClaim(ctx context.Context, q db.Querier, hostname, workspaceID string, nowMs int64) error {
	claim, err := q.FindVerifiedDomainClaimExcludingWorkspace(ctx, db.FindVerifiedDomainClaimExcludingWorkspaceParams{
		Domain:      hostname,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return nil
		}
		return fault.Wrap(err, fault.Internal("failed to find contested claim"))
	}

	logger.Info("revoking domain from previous workspace",
		"domain", hostname,
		"source", claim.Source,
		"old_workspace", claim.WorkspaceID,
		"new_workspace", workspaceID,
	)

	reason := sql.NullString{Valid: true, String: revokedReason}
	updatedAt := sql.NullInt64{Valid: true, Int64: nowMs}

	switch ClaimSource(claim.Source) {
	case ClaimSourceCustom:
		err = q.UpdateCustomDomainFailed(ctx, db.UpdateCustomDomainFailedParams{
			ID:                 claim.ID,
			VerificationStatus: db.CustomDomainsVerificationStatusFailed,
			VerificationError:  reason,
			UpdatedAt:          updatedAt,
		})
	case ClaimSourcePortal:
		err = q.UpdatePortalDomainFailed(ctx, db.UpdatePortalDomainFailedParams{
			ID:                 claim.ID,
			VerificationStatus: db.PortalDomainsVerificationStatusFailed,
			VerificationError:  reason,
			UpdatedAt:          updatedAt,
		})
	default:
		return fault.New(fmt.Sprintf("unknown domain claim source %q", claim.Source))
	}
	if err != nil {
		return fault.Wrap(err, fault.Internal("failed to mark contested claim failed"))
	}

	if err := q.DeleteFrontlineRouteByFQDN(ctx, hostname); err != nil && !db.IsNotFound(err) {
		return fault.Wrap(err, fault.Internal("failed to delete contested route"))
	}

	if err := q.DeleteAcmeChallengeByDomainID(ctx, claim.ID); err != nil && !db.IsNotFound(err) {
		return fault.Wrap(err, fault.Internal("failed to delete contested ACME challenge"))
	}

	return nil
}
