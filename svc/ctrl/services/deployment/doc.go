// Package deployment provides the control-plane deployment service.
//
// This is a ConnectRPC service on the internal control API, authenticated with
// a preshared bearer token. It is what callers that are not the public API use
// to reach a deployment: the dashboard authorizes and cancels through it, the
// dashboard's Stripe webhook deprovisions Compute through it, and the ops
// service rebuilds through it. Public-API deployment lifecycle calls (stop,
// start, promote, rollback) do not come here at all; they go straight to the
// worker's Restate handlers.
//
// # What lives here
//
//   - [Service.GetDeployment] reads a deployment and its instances.
//   - [Service.AuthorizeDeployment] approves a deployment that is waiting for a
//     maintainer, dispatches the build, and replaces the blocking GitHub commit
//     status.
//   - [Service.CancelDeployment] aborts a build in flight.
//   - [Service.DeprovisionCompute] tears down a workspace's compute and clears
//     its Deploy entitlement.
//   - [Service.Rebuild] is a plain method, not an RPC: the ops service wraps it.
//
// Rows are never written here. Every deployment row is created by the worker's
// DeployService.Create, so this service validates, changes status, and hands
// work to Restate.
//
// # Concurrency model
//
// Restate invocations are keyed by deployment_id, so deployments in the same
// project and environment build in parallel. The one contended resource
// (apps.current_deployment_id) is serialized inside RoutingService, which is
// keyed by environment.
//
// # Error handling
//
// Methods return Connect codes: [connect.CodeInvalidArgument] for validation,
// [connect.CodeNotFound] for missing resources,
// [connect.CodeFailedPrecondition] for a state or billing gate that refuses the
// action, and [connect.CodeInternal] for system failures.
package deployment
