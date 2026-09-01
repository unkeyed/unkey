package environment

import (
	"fmt"

	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/audit"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/deploycancel"
)

// envDeletedMessage is stamped onto in-flight deployment steps when an
// environment is being deleted. The environment (and its deployment views)
// are gone by the time anyone could look, so this is never user-visible.
const envDeletedMessage = "Environment deleted"

// Delete removes an environment and all associated resources.
//
// In-flight deployments are cancelled first so the cascade below doesn't
// drop deployment rows out from under workflows that are still mid-build.
// This handler is the single chokepoint for deployment row deletion;
// project and app deletes fan out to here via the virtual object cascade.
//
// Key: environment_id.
func (s *Service) Delete(
	ctx restate.ObjectContext,
	req *hydrav1.DeleteEnvironmentRequest,
) (*hydrav1.DeleteEnvironmentResponse, error) {
	envID := restate.Key(ctx)

	logger.Info("starting environment deletion", "environment_id", envID)

	// Capture env metadata before the row is deleted, for the audit log.
	env, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.Environment, error) {
		return s.db.FindEnvironmentById(runCtx, envID)
	}, restate.WithName("find environment"))
	if err != nil {
		return nil, fmt.Errorf("find environment: %w", err)
	}

	if err := s.cancelProgressingDeployments(ctx, env, req); err != nil {
		return nil, fmt.Errorf("cancel progressing deployments: %w", err)
	}

	if err := s.cancelDomainVerifications(ctx, envID); err != nil {
		return nil, fmt.Errorf("cancel domain verifications: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return s.db.DeleteCiliumNetworkPoliciesByEnvironmentId(runCtx, envID)
	}, restate.WithName("delete network policies")); err != nil {
		return nil, fmt.Errorf("delete network policies: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return s.db.DeleteCustomDomainsByEnvironmentId(runCtx, envID)
	}, restate.WithName("delete custom domains")); err != nil {
		return nil, fmt.Errorf("delete custom domains: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return s.db.DeleteFrontlineRoutesByEnvironmentId(runCtx, envID)
	}, restate.WithName("delete frontline routes")); err != nil {
		return nil, fmt.Errorf("delete frontline routes: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return s.db.DeleteAppEnvVarsByEnvironmentId(runCtx, envID)
	}, restate.WithName("delete env vars")); err != nil {
		return nil, fmt.Errorf("delete env vars: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return s.db.DeleteAppRegionalSettingsByEnvironmentId(runCtx, envID)
	}, restate.WithName("delete regional settings")); err != nil {
		return nil, fmt.Errorf("delete regional settings: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return s.db.DeleteAppBuildSettingsByEnvironmentId(runCtx, envID)
	}, restate.WithName("delete build settings")); err != nil {
		return nil, fmt.Errorf("delete build settings: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return s.db.DeleteAppRuntimeSettingsByEnvironmentId(runCtx, envID)
	}, restate.WithName("delete runtime settings")); err != nil {
		return nil, fmt.Errorf("delete runtime settings: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return s.db.DeleteDeploymentStepsByEnvironmentId(runCtx, envID)
	}, restate.WithName("delete deployment steps")); err != nil {
		return nil, fmt.Errorf("delete deployment steps: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return s.db.DeleteDeploymentTopologiesByEnvironmentId(runCtx, envID)
	}, restate.WithName("delete deployment topologies")); err != nil {
		return nil, fmt.Errorf("delete deployment topologies: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return s.db.DeleteDeploymentsByEnvironmentId(runCtx, envID)
	}, restate.WithName("delete deployments")); err != nil {
		return nil, fmt.Errorf("delete deployments: %w", err)
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return s.db.DeleteEnvironmentById(runCtx, envID)
	}, restate.WithName("delete environment")); err != nil {
		return nil, fmt.Errorf("delete environment: %w", err)
	}

	// The environment has no display name, so its slug stands in.
	if err := audit.Insert(ctx, s.auditlogs, audit.Event{
		Actor:         req.GetActor(),
		CorrelationID: req.GetCorrelationId(),
		WorkspaceID:   env.WorkspaceID,
		Event:         auditlog.EnvironmentDeleteEvent,
		Display:       fmt.Sprintf("Deleted environment %s", env.Slug),
		Resource: auditlog.AuditLogResource{
			ID:          env.ID,
			Type:        auditlog.EnvironmentResourceType,
			Meta:        map[string]any{"slug": env.Slug, "appId": env.AppID, "projectId": env.ProjectID},
			Name:        env.Slug,
			DisplayName: env.Slug,
		},
	}); err != nil {
		return nil, fmt.Errorf("insert audit log: %w", err)
	}

	logger.Info("environment deletion complete", "environment_id", envID)

	return &hydrav1.DeleteEnvironmentResponse{}, nil
}

// cancelProgressingDeployments aborts in-flight deployments through
// deploycancel.Cancel, audited per deployment with the deletion's actor.
//
// The whole call is one journaled step. That is what makes it safe for the
// helper to flip statuses before cancelling invocations: the row set was
// journaled by the list step, so a retry after a failed cancel re-runs the
// helper against the SAME rows rather than re-listing and skipping the ones
// already flipped terminal. A cancel failure therefore retries until every
// invocation is dead, and the audit entries (written last inside the helper)
// land exactly once, on the pass where nothing fails anymore.
func (s *Service) cancelProgressingDeployments(
	ctx restate.ObjectContext,
	env db.Environment,
	req *hydrav1.DeleteEnvironmentRequest,
) error {
	active, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]db.ListProgressingDeploymentsByEnvironmentIdRow, error) {
		return s.db.ListProgressingDeploymentsByEnvironmentId(runCtx, db.ListProgressingDeploymentsByEnvironmentIdParams{
			EnvironmentID:       env.ID,
			ProgressingStatuses: mysqltype.ProgressingDeploymentStatuses,
		})
	}, restate.WithName("list progressing deployments"))
	if err != nil {
		return fmt.Errorf("list progressing deployments: %w", err)
	}

	if len(active) == 0 {
		return nil
	}

	targets := make([]deploycancel.Target, 0, len(active))
	for _, d := range active {
		invocationID := ""
		if d.InvocationID.Valid {
			invocationID = d.InvocationID.String
		}
		targets = append(targets, deploycancel.Target{ID: d.ID, InvocationID: invocationID})
	}

	logger.Info("cancelling in-flight deployments for environment deletion",
		"environment_id", env.ID,
		"count", len(targets),
	)

	return restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return deploycancel.Cancel(runCtx, s.db, s.admin, deploycancel.Params{
			Targets: targets,
			Reason:  envDeletedMessage,
			Status:  mysqltype.DeploymentsStatusCancelled,
			Audit: &deploycancel.Audit{
				Service:       s.auditlogs,
				Actor:         req.GetActor(),
				CorrelationID: req.GetCorrelationId(),
				WorkspaceID:   env.WorkspaceID,
				Meta: map[string]any{
					"projectId":     env.ProjectID,
					"appId":         env.AppID,
					"environmentId": env.ID,
				},
			},
		})
	}, restate.WithName("cancel progressing deployments"))
}

// cancelDomainVerifications aborts in-flight custom domain verification
// workflows before the cascade deletes custom_domains rows. Without this,
// VerifyDomain retries would keep hitting sql.ErrNoRows until the 24-hour
// retry window expires.
func (s *Service) cancelDomainVerifications(ctx restate.ObjectContext, envID string) error {
	domains, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]db.CustomDomain, error) {
		return s.db.ListCustomDomainsByEnvironmentID(runCtx, envID)
	}, restate.WithName("list custom domains"))
	if err != nil {
		return fmt.Errorf("list custom domains: %w", err)
	}

	if len(domains) == 0 {
		return nil
	}

	logger.Info("cancelling in-flight domain verifications for environment deletion",
		"environment_id", envID,
		"count", len(domains),
	)

	for _, d := range domains {
		if !d.InvocationID.Valid || d.InvocationID.String == "" {
			continue
		}

		invocationID := d.InvocationID.String
		domainID := d.ID

		if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
			return s.admin.CancelInvocation(runCtx, invocationID)
		}, restate.WithName("cancel domain verification "+domainID)); err != nil {
			logger.Warn("failed to cancel domain verification workflow",
				"environment_id", envID,
				"domain_id", domainID,
				"domain", d.Domain,
				"invocation_id", invocationID,
				"error", err,
			)
		}
	}

	return nil
}
