// Package github provides the GitHub webhook HTTP handler for the control plane.
// It receives webhook events from GitHub, verifies signatures, and triggers
// the GitHubService Restate workflow for durable processing.
package github
