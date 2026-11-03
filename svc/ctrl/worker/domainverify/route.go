package domainverify

import (
	"context"
	"database/sql"
	"fmt"

	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// routeConflictReason is what the tenant reads on a row whose hostname is
// already routed elsewhere.
const routeConflictReason = "domain is already routed to another target"

// Route is the frontline route a verified hostname row gets. DomainID is the
// row's id, in whichever table it lives.
type Route struct {
	DomainID      string
	Hostname      string
	ProjectID     string
	AppID         string
	EnvironmentID string
	// DeploymentID is empty when the app has no deployment yet; the route is
	// assigned when the first one lands.
	DeploymentID string
}

// CreateRoute inserts route with sticky=live so it follows the app's deploys.
//
// The insert generates a fresh route id each attempt, so a retry of an insert
// that already committed hits this row's own route, which counts as success.
// A route owned by anything else can never be inserted, so markFailed records
// the conflict on the row, its ACME challenge is removed, and the returned
// terminal error stops retries instead of waiting out the 24-hour window.
func CreateRoute(
	ctx context.Context,
	q db.Querier,
	route Route,
	nowMs int64,
	markFailed func(ctx context.Context, reason string) error,
) error {
	err := q.InsertFrontlineRoute(ctx, db.InsertFrontlineRouteParams{
		ID:                       uid.New(uid.FrontlineRoutePrefix),
		ProjectID:                route.ProjectID,
		AppID:                    route.AppID,
		DeploymentID:             route.DeploymentID,
		EnvironmentID:            route.EnvironmentID,
		FullyQualifiedDomainName: route.Hostname,
		Sticky:                   db.FrontlineRoutesStickyLive,
		CreatedAt:                nowMs,
		UpdatedAt:                sql.NullInt64{Valid: true, Int64: nowMs},
	})
	if err == nil {
		return nil
	}
	if !db.IsDuplicateKeyError(err) {
		return fault.Wrap(err, fault.Internal("failed to insert frontline route"))
	}

	existing, err := q.FindFrontlineRouteByFQDN(ctx, route.Hostname)
	if err != nil {
		return fault.Wrap(err, fault.Internal("failed to read conflicting frontline route"))
	}
	if existing.ProjectID == route.ProjectID &&
		existing.EnvironmentID == route.EnvironmentID &&
		existing.FullyQualifiedDomainName == route.Hostname {
		return nil
	}

	if err := markFailed(ctx, routeConflictReason); err != nil {
		return fault.Wrap(err, fault.Internal("failed to mark domain as failed"))
	}
	if err := q.DeleteAcmeChallengeByDomainID(ctx, route.DomainID); err != nil && !db.IsNotFound(err) {
		return fault.Wrap(err, fault.Internal("failed to delete ACME challenge"))
	}

	logger.Warn("frontline route conflict, domain marked failed",
		"domain", route.Hostname,
		"domain_id", route.DomainID,
		"route_id", existing.ID,
	)
	return restate.ToTerminalError(fmt.Errorf("frontline route conflict for %s: %s", route.Hostname, routeConflictReason))
}
