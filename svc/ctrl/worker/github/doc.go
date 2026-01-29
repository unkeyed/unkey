// Package github implements the GitHubService Restate workflow for handling
// GitHub App webhook events. It orchestrates the deployment flow from
// GitHub push events through tarball download, S3 upload, and deployment
// creation using the existing DeploymentService workflow.
package github
