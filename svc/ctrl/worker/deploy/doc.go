// Package deploy orchestrates the deployment lifecycle for user applications.
//
// Deployments move through a multi-step pipeline that builds container images,
// provisions infrastructure across regions, waits for health, and configures
// domain routing — all durably, so a crash at any point resumes from the last
// completed step rather than restarting from scratch.
//
// # Virtual Object Keying
//
// DeployService is a Restate virtual object keyed by deployment_id. Each
// deployment runs as its own isolated workflow, so multiple deployments per
// environment can build in parallel. The contended resource
// (apps.current_deployment_id) is serialized inside RoutingService via
// SwapLiveDeployment, which is keyed by env_id.
//
// Workspace-wide concurrency is capped by [buildslot.Service].
//
// # Why Restate Workflows
//
// Every handler in this package runs as a Restate Workflow (restate.dev), keyed
// as described above. Restate provides automatic retries, durable execution
// across process restarts, and exactly-once semantics per step. Each step is
// wrapped in restate.Run (or RunAsync for parallel work), making the entire
// pipeline resumable. If the process dies mid-deploy, Restate picks up from
// the last committed step without re-executing earlier side effects.
//
// # Build Queue and Dedup
//
// Before starting the actual build, [Workflow.Deploy] goes through two gates:
//
//  1. Self-skip: [Workflow.skipIfSuperseded] checks
//     [db.Queries.HasNewerActiveDeployment] for a newer sibling on the same
//     (app, env, branch). If one exists in any non-terminal status, this
//     deployment marks itself as skipped and returns.
//  2. Concurrency gate: [Workflow.waitForBuildSlot] creates a Restate
//     awakeable and calls [hydrav1.BuildSlotService.AcquireOrWait]. The
//     handler parks on the awakeable until BuildSlotService resolves it —
//     either immediately (slot available or is_production=true) or later when
//     a held slot is released. Production deployments bypass the limit.
//
// On the creation side, [dedup.CancelOlderSiblings] runs right after the
// deployment row is inserted: it batch-stamps older siblings with the
// "Superseded by newer commit" marker, batch-transitions them to
// status=superseded, and cancels their Restate invocations via the admin API.
//
// # Operations
//
// [Workflow.Deploy] is the primary entrypoint. It validates the deployment
// record, loads workspace/project/environment context, then either builds a
// Docker image from a Git repository via Depot or accepts a pre-built image.
// It creates deployment topologies for every configured region (each with its
// own deployment_changes entry), ensures sentinel containers and Cilium
// network policies exist per region, and polls in parallel until all instances
// are running. New sentinels are tracked via [SentinelService.Deploy], which
// blocks until the sentinel has converged in Kubernetes before proceeding.
// Existing sentinels are not auto-upgraded during deploys; image rollouts
// are a separate operation.
// Once healthy, it generates frontline routes for per-commit,
// per-branch, and per-environment domains, reassigns sticky routes through
// RoutingService, marks the deployment ready, and — for non-rolled-back
// production environments — updates the app's live deployment pointer.
// The previous live deployment is scheduled for standby after 30 minutes via
// DeploymentService.ScheduleDesiredStateChange.
//
// [Workflow.Rollback] switches sticky frontline routes (environment and live)
// from the current live deployment to a previous one, atomically through
// RoutingService. Because the live pointer now points to an older deployment,
// subsequent deploys detect the rolled-back state and skip auto-promotion.
//
// [Workflow.Promote] reassigns sticky routes to a new target deployment and
// updates the live pointer, restoring normal auto-promote behavior for future
// deploys.
//
// [Workflow.ScaleDownIdlePreviewDeployments] paginates through preview
// environments and sets any deployment to archived that has been idle (zero
// requests in ClickHouse) for longer than six hours.
//
// # Instance Readiness
//
// [Workflow.waitForDeployments] parks on a Restate awakeable instead of
// polling. It stores {awakeable_id, deployment_id} in the VO state under
// [instancesReadyAwakeableKey], does an initial health check (resolving
// the awakeable immediately if instances are already healthy), then races
// the awakeable against [regionReadyTimeout] via [restate.WaitFirst].
//
// [Workflow.NotifyInstancesReady] is a SHARED handler called by
// services/cluster's ReportDeploymentStatus RPC after it detects that
// enough regions have become healthy. It verifies the request's
// deployment_id matches the one in state (protecting against late reports
// from previous deployments on the same VO key) and then resolves the
// awakeable.
//
// Sentinel convergence and pod readiness run in parallel: the Deploy
// handler fires [Workflow.fanOutSentinelDeploys] to kick off
// SentinelService.Deploy futures, then enters waitForDeployments. Krane
// works on both concurrently; [Workflow.waitForSentinels] collects
// the sentinel futures after the pod wait returns.
//
// # Cancellation
//
// Users can manually cancel an in-flight deployment via the CancelDeployment
// RPC on the control API ([services/deployment.Service.CancelDeployment]).
// The RPC stamps any active deployment steps with "Cancelled by user" (via
// [db.Queries.EndActiveDeploymentStepsWithError]) and calls
// [restateadmin.Client.CancelInvocation] on the stored invocation_id. Restate
// injects a TerminalError at the handler's next SDK call, which triggers the
// deferred compensation stack to release the build slot, mark the deployment
// as failed (via the conditional [db.Queries.UpdateDeploymentStatusIfActive]
// which never overwrites terminal statuses), and unwind partial state.
//
// Sibling cancellation (dedup) uses the same mechanism but stamps
// "Superseded by newer commit" and transitions the status to superseded.
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
// [buildDomains] produces four or five domain patterns per deployment:
//
//   - Per-commit: <project>-<app>-git-<sha>-<workspace>.<apex>  (non-sticky, immutable)
//   - Per-branch: <project>-<app>-git-<branch>-<workspace>.<apex>  (sticky to branch)
//   - Per-environment: <project>-<app>-<env>-<workspace>.<apex>  (sticky to environment)
//   - Per-live (production only): <project>-<app>-<workspace>.<apex>  (sticky to live)
//   - Per-deployment: <project>-<app>-dep-<id>-<workspace>.<apex>  (sticky to deployment)
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
