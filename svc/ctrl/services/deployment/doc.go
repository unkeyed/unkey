// Package deployment is the ConnectRPC surface for acting on a deployment that
// already exists. Rows are only written by hydra.DeployWorkflow.Create in the
// deploy worker.
//
//   - [Service.GetDeployment] reads one.
//   - [Service.AuthorizeDeployment] lets a project member approve a fork PR build.
//   - [Service.CancelDeployment] aborts one in flight.
//   - [Service.Rebuild] reproduces one from its source. The ops service calls it.
//   - [Service.DeprovisionCompute] tears down a workspace's compute when its
//     Compute plan is cancelled.
//
// Each method validates and authorizes, then hands the durable work to Restate:
// authorize sends Deploy, rebuild calls Create, deprovision sends Teardown.
// Rebuild alone waits for its callee so an operator sees a rejection at once.
//
// Connect codes follow the usual convention: [connect.CodeInvalidArgument] for a
// malformed request, [connect.CodeNotFound] for a deployment the caller cannot
// see, [connect.CodeFailedPrecondition] for a state that forbids the action, and
// [connect.CodeInternal] for a failure the caller cannot act on.
package deployment
