// Package dedup centralises the "cancel in-progress siblings" logic used
// when a new deployment is created for a branch that already has an
// active build. A fresh commit supersedes the current build instead of
// queueing behind it.
package dedup

import (
	"context"
	"database/sql"
	"fmt"

	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"

	"github.com/unkeyed/unkey/pkg/logger"
	restateadmin "github.com/unkeyed/unkey/pkg/restate/admin"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/deploycancel"
)

// SupersededByNewerCommitMessage is stamped onto the in-flight deployment
// step of a sibling that is being cancelled because a newer commit for the
// same branch landed. The frontend matches on this exact string to render
// the superseded deployment with a dedicated view instead of the red
// FailedDeploymentBanner. Must stay in sync with SUPERSEDED_BY_NEWER in
// web/apps/dashboard/.../cancelled-marker.ts.
const SupersededByNewerCommitMessage = "Superseded by newer commit"

// Service handles cancellation of superseded sibling deployments.
type Service struct {
	db    db.Database
	admin *restateadmin.Client
}

// New creates a dedup Service.
func New(database db.Database, admin *restateadmin.Client) *Service {
	return &Service{db: database, admin: admin}
}

// Newer identifies the deployment that triggered sibling cancellation.
type Newer struct {
	ID            string
	AppID         string
	EnvironmentID string
	GitBranch     string
	CreatedAt     int64
}

// CancelOlderSiblings finds deployments for the same (app, environment,
// branch) that were created before newer and are still in the build queue
// (status `pending` or `awaiting_approval` -- haven't acquired a build slot
// yet), then hands them to deploycancel: superseded status, no audit entries
// (machine-initiated, the customer never asked for it).
//
// Once a deployment transitions out of `pending` (acquired a build slot and
// moved to `starting`/`building`/etc), it is committed and will not be
// superseded by a newer commit. This avoids the pathological "rapid pushes
// keep cancelling builds and nothing ever finishes" scenario.
//
// Best-effort: failed invocation cancels don't stop the remaining siblings;
// they come back joined in the returned error for the caller to log. The rows
// are superseded either way, and Deploy refuses to build a terminal row, so a
// missed cancel cannot resurrect a sibling.
//
// Only git-sourced deployments with a branch are deduplicated -- docker
// image redeploys are manual and should never cancel siblings.
func (s *Service) CancelOlderSiblings(ctx context.Context, newer Newer) error {
	if newer.GitBranch == "" {
		return nil
	}

	older, err := s.db.ListOlderActiveDeploymentsForDedup(ctx, db.ListOlderActiveDeploymentsForDedupParams{
		AppID:         newer.AppID,
		EnvironmentID: newer.EnvironmentID,
		GitBranch:     sql.NullString{Valid: true, String: newer.GitBranch},
		CreatedAt:     newer.CreatedAt,
		DeploymentID:  newer.ID,
	})
	if err != nil {
		return fmt.Errorf("list older active deployments: %w", err)
	}

	if len(older) == 0 {
		return nil
	}

	logger.Info("cancelling superseded sibling deployments",
		"count", len(older),
		"newer_deployment_id", newer.ID,
		"app_id", newer.AppID,
		"environment_id", newer.EnvironmentID,
		"branch", newer.GitBranch,
	)

	targets := make([]deploycancel.Target, 0, len(older))
	for _, old := range older {
		invocationID := ""
		if old.InvocationID.Valid {
			invocationID = old.InvocationID.String
		}
		targets = append(targets, deploycancel.Target{ID: old.ID, InvocationID: invocationID})
	}

	// The nil check cannot move into the helper: a nil *Client in a non-nil
	// interface value would pass its admin != nil guard and panic.
	var canceler deploycancel.InvocationCanceler
	if s.admin != nil {
		canceler = s.admin
	}

	return deploycancel.Cancel(ctx, s.db, canceler, deploycancel.Params{
		Targets: targets,
		Reason:  SupersededByNewerCommitMessage,
		Status:  mysqltype.DeploymentsStatusSuperseded,
		Audit:   nil,
	})
}
