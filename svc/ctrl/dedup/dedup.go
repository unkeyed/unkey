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

// SupersededByNewerCommitMessage is written to deployment_steps.error on the
// open step of an older deployment that is cancelled because a newer commit
// landed on the same branch. The dashboard shows this text on that step. The
// dashboard chooses its superseded view from deployments.status, not from this
// string, so changing the text is safe.
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

// CancelOlderSiblings moves older deployments on the same app, environment, and
// branch to status superseded through deploycancel.Cancel. Only rows whose
// status is pending or awaiting_approval qualify. A deployment that reached
// starting already holds a build slot and is left alone; otherwise rapid pushes
// could keep cancelling builds and never ship one.
//
// No audit entries are written: this cancel is started by the system and has
// no user to attribute it to.
//
// When admin.CancelInvocation fails, the sibling's row is already superseded.
// If its Workflow.Deploy has not yet reached the status check at its top, it
// stops there. If it is already parked in Workflow.waitForBuildSlot, it keeps
// running and resumes when a slot frees.
//
// Deployments without a branch, such as image redeploys, are never
// deduplicated.
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

	// Assigning a nil *restateadmin.Client to the interface makes a non-nil
	// interface holding a nil pointer. deploycancel.Cancel only checks
	// admin == nil, so it would call CancelInvocation on the nil pointer and
	// panic.
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
