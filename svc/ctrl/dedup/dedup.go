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

// CancelOlderSiblings supersedes queued deployments (pending or
// awaiting_approval) on the same app, environment, and branch that were created
// before newer. A deployment that already holds a build slot is left to finish,
// so rapid pushes cannot keep cancelling builds and never ship one.
//
// No audit entries: the cancel is machine-initiated and has no actor. A failed
// invocation cancel comes back in the error but cannot resurrect a sibling,
// because the row is already superseded and Deploy refuses a terminal row.
//
// Deployments without a branch (image redeploys) are never deduplicated.
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

	// A nil *Client stored in the interface is not a nil interface, so
	// deploycancel would call it and panic.
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
