// Package deployment is the ConnectRPC DeployService. Its one method,
// [Service.DeprovisionCompute], tears down a workspace's compute when its
// Compute plan is cancelled: it sends Teardown to Restate and clears the
// workspace's deploy plan.
//
// Connect codes follow the usual convention: [connect.CodeInvalidArgument] for a
// malformed request, [connect.CodeNotFound] for a workspace that does not exist,
// and [connect.CodeInternal] for a failure the caller cannot act on.
package deployment
