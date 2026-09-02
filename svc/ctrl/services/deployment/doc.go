// Package deployment is the ConnectRPC surface for acting on a deployment that
// already exists.
//
// Creating deployments moved to hydra.DeployService.Create in the deploy
// worker, which is the only writer of deployment rows. What remains here are
// the operations a caller performs against a deployment by id: reading it
// ([Service.GetDeployment]), letting a project member approve a fork PR's build
// ([Service.AuthorizeDeployment]), aborting one in flight
// ([Service.CancelDeployment]), reproducing one ([Service.Rebuild], reached
// through the ops service), and tearing a workspace's compute down when it
// cancels its Compute plan ([Service.DeprovisionCompute]).
//
// Durable work is not done here. Each method validates and authorizes, then
// hands off to Restate: authorize sends Deploy, rebuild sends Create, and
// deprovision sends Teardown. Restate keys those on the deployment or workspace
// id, so this service holds no locks of its own and two calls about different
// deployments never contend.
//
// [Service.Rebuild] is the exception that waits for its callee. It uses Request
// rather than Send so an operator learns immediately that a rebuild was refused,
// which is why it maps a BLOCKED outcome onto [connect.CodeFailedPrecondition].
//
// Methods return Connect codes by the usual convention:
// [connect.CodeInvalidArgument] for a malformed request,
// [connect.CodeNotFound] for a deployment the caller cannot see,
// [connect.CodeFailedPrecondition] for a state that forbids the action, and
// [connect.CodeInternal] for a failure the caller cannot act on.
package deployment
