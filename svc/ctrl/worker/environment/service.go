package environment

import (
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/auditlog"
	restateadmin "github.com/unkeyed/unkey/pkg/restate/admin"
	"github.com/unkeyed/unkey/svc/ctrl/internal/audit"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auditlogs"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// standbyDelay keeps a demoted deployment running long enough to roll back to
// it without a cold start.
const standbyDelay = 30 * time.Minute

// runMaxAttempts bounds per-Run retries so a persistent failure returns a
// terminal error instead of eating the invocation retry budget.
const runMaxAttempts uint = 5

// Service implements the EnvironmentService Restate virtual object. The key is
// the environment ID, so deletion, promotion, and rollback of one environment
// never overlap.
type Service struct {
	hydrav1.UnimplementedEnvironmentServiceServer
	db        db.Database
	admin     *restateadmin.Client
	auditlogs auditlogs.AuditLogService
}

var _ hydrav1.EnvironmentServiceServer = (*Service)(nil)

// Config holds configuration for creating a [Service].
type Config struct {
	DB db.Database

	// Admin cancels in-flight deployment invocations before the env delete
	// cascade drops deployment rows. Required.
	Admin *restateadmin.Client

	// Auditlogs writes the environment.delete event as a durable step inside
	// the deletion workflow. Environment deletes are cascade-only, so the event
	// always carries the actor and correlation ID of the parent teardown.
	Auditlogs auditlogs.AuditLogService
}

// New creates a [Service] with the given configuration.
func New(cfg Config) (*Service, error) {
	if err := assert.All(
		assert.NotNil(cfg.Admin, "Admin must not be nil"),
		assert.NotNil(cfg.Auditlogs, "Auditlogs must not be nil"),
	); err != nil {
		return nil, err
	}
	return &Service{
		UnimplementedEnvironmentServiceServer: hydrav1.UnimplementedEnvironmentServiceServer{},
		db:                                    cfg.DB,
		admin:                                 cfg.Admin,
		auditlogs:                             cfg.Auditlogs,
	}, nil
}

// loadDeployments reads each deployment with its environment kind and its
// app's live pointer, keyed by id, in one step. A deployment outside the keyed
// environment is refused: the key is the lock.
func (s *Service) loadDeployments(ctx restate.ObjectContext, deploymentIDs ...string) (map[string]db.FindDeploymentWithEnvironmentAndAppRow, error) {
	environmentID := restate.Key(ctx)

	deployments, err := restate.Run(ctx, func(runCtx restate.RunContext) (map[string]db.FindDeploymentWithEnvironmentAndAppRow, error) {
		byID := make(map[string]db.FindDeploymentWithEnvironmentAndAppRow, len(deploymentIDs))
		for _, id := range deploymentIDs {
			row, err := s.db.FindDeploymentWithEnvironmentAndApp(runCtx, id)
			if err != nil {
				if db.IsNotFound(err) {
					return nil, restate.TerminalError(fmt.Errorf("deployment not found: %s", id), 404)
				}
				return nil, fmt.Errorf("load deployment %s: %w", id, err)
			}
			if err := assert.Equal(row.EnvironmentID, environmentID, "deployment must belong to the keyed environment"); err != nil {
				return nil, restate.TerminalError(err, 400)
			}
			byID[id] = row
		}
		return byID, nil
	}, restate.WithName("load deployments"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		return nil, err
	}
	return deployments, nil
}

// findStickyRouteIDs returns the environment and live routes. None is a caller
// error: there is nothing to move.
func (s *Service) findStickyRouteIDs(ctx restate.ObjectContext, environmentID string) ([]string, error) {
	routeIDs, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]string, error) {
		routes, err := s.db.FindFrontlineRoutesByEnvironmentAndSticky(runCtx, db.FindFrontlineRoutesByEnvironmentAndStickyParams{
			EnvironmentID: environmentID,
			Sticky: []db.FrontlineRoutesSticky{
				db.FrontlineRoutesStickyLive,
				db.FrontlineRoutesStickyEnvironment,
			},
		})
		if err != nil {
			return nil, err
		}
		ids := make([]string, 0, len(routes))
		for _, route := range routes {
			ids = append(ids, route.ID)
		}
		return ids, nil
	}, restate.WithName("find sticky routes"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		return nil, fmt.Errorf("find sticky routes: %w", err)
	}

	if len(routeIDs) == 0 {
		return nil, restate.TerminalError(fmt.Errorf("environment %s has no sticky routes", environmentID), 400)
	}
	return routeIDs, nil
}

// insertLifecycleAudit writes one lifecycle audit entry as a durable step. A
// nil actor is a retained ctrl RPC that writes its own entry; skip it.
func (s *Service) insertLifecycleAudit(
	ctx restate.ObjectContext,
	actor *ctrlv1.ActorInfo,
	correlationID string,
	deployment db.FindDeploymentWithEnvironmentAndAppRow,
	event auditlog.AuditLogEvent,
	display string,
) error {
	if actor == nil {
		return nil
	}

	return audit.Insert(ctx, s.auditlogs, audit.Event{
		Actor:         actor,
		CorrelationID: correlationID,
		WorkspaceID:   deployment.WorkspaceID,
		Event:         event,
		Display:       display,
		Resource: auditlog.AuditLogResource{
			Type:        auditlog.DeploymentResourceType,
			ID:          deployment.ID,
			Name:        "",
			DisplayName: deployment.ID,
			Meta: map[string]any{
				"projectId":     deployment.ProjectID,
				"appId":         deployment.AppID,
				"environmentId": deployment.EnvironmentID,
			},
		},
	})
}
