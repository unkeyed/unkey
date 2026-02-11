// Package deploy orchestrates the deployment lifecycle for user applications.
//
// Deployments move through a multi-step pipeline that builds container images,
// provisions infrastructure across regions, waits for health, and configures
// domain routing — all durably, so a crash at any point resumes from the last
// completed step rather than restarting from scratch.
//
// # Why Restate Workflows
//
// Every handler in this package runs as a Restate Workflow (restate.dev), keyed
// by a caller-supplied workflow ID. Restate provides automatic retries, durable
// execution across process restarts, and exactly-once semantics per step. Each
// step is wrapped in restate.Run (or RunAsync for parallel work), making the
// entire pipeline resumable. If the process dies mid-deploy, Restate picks up
// from the last committed step without re-executing earlier side effects.
//
// # Operations
//
// [Workflow.Deploy] is the primary entrypoint. It validates the deployment
// record, loads workspace/project/environment context, then either builds a
// Docker image from a Git repository via Depot or accepts a pre-built image.
// It creates deployment topologies for every configured region (each with its
// own version from VersioningService), ensures sentinel containers and Cilium
// network policies exist per region, and polls in parallel until all instances
// are running. Once healthy, it generates frontline routes for per-commit,
// per-branch, and per-environment domains, reassigns sticky routes through
// RoutingService, marks the deployment ready, and — for non-rolled-back
// production environments — updates the project's live deployment pointer.
// The previous live deployment is scheduled for standby after 30 minutes via
// DeploymentService.ScheduleDesiredStateChange.
//
// [Workflow.Rollback] switches sticky frontline routes (environment and live)
// from the current live deployment to a previous one, atomically through
// RoutingService. It sets the project's isRolledBack flag, which prevents
// subsequent deploys from automatically claiming the live routes.
//
// [Workflow.Promote] reverses a rollback by reassigning sticky routes to a new
// target deployment and clearing the isRolledBack flag so normal deploy flow
// resumes.
//
// [Workflow.ScaleDownIdlePreviewDeployments] paginates through preview
// environments and sets any deployment to archived that has been idle (zero
// requests in ClickHouse) for longer than six hours.
//
// # Image Builds
//
// When a deploy request carries a Git source, the workflow builds a container
// image remotely through Depot. It retrieves or creates a Depot project per
// Unkey project, acquires a BuildKit machine, fetches the repository via a
// GitHub installation token, and streams build-step telemetry to ClickHouse.
//
// # Domain Generation
//
// [buildDomains] produces three domain patterns per deployment:
//
//   - Per-commit: <project>-git-<sha>-<workspace>.<apex>  (non-sticky, immutable)
//   - Per-branch: <project>-git-<branch>-<workspace>.<apex>  (sticky to branch)
//   - Per-environment: <project>-<env>-<workspace>.<apex>  (sticky to environment)
//
// Sticky domains automatically follow the latest deployment matching their
// criteria; commit domains never move. CLI uploads add a random numeric suffix
// to the commit domain to avoid collisions from repeated pushes of the same
// SHA. Live-traffic routing is handled separately by sticky route reassignment
// in RoutingService, not by a dedicated "live" domain type.
//
// # Network Policy
//
// [Workflow.ensureCiliumNetworkPolicy] persists Cilium network policies in the
// database for each region that lacks one. Each policy allows ingress from the
// sentinel namespace to deployment pods on port 8080 and is applied by regional
// reconcilers.
//
// # Error Handling
//
// Deploy defers a cleanup handler that marks the deployment as failed if the
// workflow does not finish successfully. Terminal errors (invalid input, not
// found) are returned with appropriate HTTP status codes so Restate does not
// retry them; transient failures are returned as regular errors for automatic
// retry.
package deploy
