// Package routing manages frontline route assignment for deployments.
//
// This package provides a Restate-based service for reassigning frontline routes
// (the edge routing layer) to point at different deployments. When a deployment
// is promoted or traffic needs to shift, this service updates the database records
// that control which deployment receives traffic for each route.
//
// # Architecture
//
// The routing service implements [hydrav1.RoutingServiceServer] and runs as a
// Restate virtual object. Virtual objects provide serialized access per key,
// preventing race conditions when multiple operations target the same routes.
//
// # Restate Integration
//
// All route reassignment operations use Restate's durable execution model.
// Each route update is wrapped in [restate.Run], which provides automatic retries
// on transient failures and exactly-once execution guarantees. If the service
// crashes mid-operation, Restate replays completed steps and resumes from where
// it left off.
//
// # Key Operations
//
// [Service.AssignFrontlineRoutes] reassigns a set of frontline routes to a new
// deployment by updating the deployment_id column in the frontline_routes table.
//
// # Usage
//
// Create a routing service:
//
//	routingSvc := routing.New(routing.Config{
//		DB:            mainDB,
//		DefaultDomain: "unkey.app",
//	})
//
// Register with Restate and invoke via the generated client to reassign routes
// during deployment promotion or rollback operations.
package routing
