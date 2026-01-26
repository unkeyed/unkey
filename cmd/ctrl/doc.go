// Package ctrl provides CLI commands for running the Unkey control plane.
//
// The control plane consists of two services that work together to manage
// Unkey's infrastructure: an API server for handling requests and a worker
// for executing background jobs. Both services are designed for distributed
// deployment and integrate with Restate for durable workflow execution.
//
// # Commands
//
// The package exposes a single [Cmd] that contains two subcommands:
//
//   - api: Runs the control plane API server
//   - worker: Runs the background job processor
//
// # API Server
//
// The api subcommand starts an HTTP server that handles control plane requests
// including infrastructure provisioning, build management, and service
// orchestration. It requires a MySQL database connection and integrates with
// S3-compatible storage for build artifacts, Vault for secrets management,
// and Restate for workflow coordination.
//
// TLS is optional but both --tls-cert-file and --tls-key-file must be provided
// together to enable HTTPS. The server validates this constraint and exits
// with an error if only one is provided.
//
// ACME support enables automatic TLS certificate provisioning via Let's Encrypt
// using Route53 DNS-01 challenges for domain validation.
//
// # Worker
//
// The worker subcommand starts a background processor that handles durable
// workflows including deployments, container builds, and certificate management.
// It supports two build backends: "docker" for local development and "depot"
// for production builds. The worker registers itself with Restate for receiving
// workflow invocations.
//
// The worker supports both Cloudflare and Route53 DNS providers for ACME
// certificate challenges, allowing flexibility based on where domains are hosted.
//
// # Configuration
//
// Both services accept configuration through CLI flags and environment variables.
// Required flags will cause the service to fail on startup if not provided.
// See the individual flag definitions in [Cmd] for defaults and environment
// variable mappings.
package ctrl
