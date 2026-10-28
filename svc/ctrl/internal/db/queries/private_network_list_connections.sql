-- name: ListPrivateNetworkConnectionSelections :many
-- ListPrivateNetworkConnectionSelections returns the directed app connections
-- saved by the given caller deployments, with the saved target selection and
-- the caller and live deployment facts that ListPrivateNetworkConnections
-- needs to resolve it. LiveEnvironmentID is the environment of the target
-- app's live deployment, empty when there is none. Names starting with unkey
-- and the name self are reserved. Callers must come from
-- ListPrivateNetworkReplicas in the same transaction, which decides which
-- deployments are active callers. Rows are paged by (caller deployment,
-- connection).
SELECT
    b.workspace_id,
    b.project_id,
    b.resource_id AS app_id,
    target_app.slug AS app_slug,
    w.k8s_namespace,
    caller.id AS caller_deployment_id,
    b.connection_id,
    b.name AS connection_name,
    t.selection_mode,
    COALESCE(t.target_deployment_id, '') AS target_deployment_id,
    COALESCE(t.target_environment_id, '') AS target_environment_id,
    caller_env.kind AS caller_environment_kind,
    caller.source AS caller_source,
    COALESCE(caller.git_branch, '') AS caller_git_branch,
    COALESCE(caller.fork_repository_full_name, '') AS caller_fork_repository_full_name,
    COALESCE(target_app.current_deployment_id, '') AS live_deployment_id,
    COALESCE(live_env.id, '') AS live_environment_id
FROM deployment_connections b
STRAIGHT_JOIN deployments caller ON caller.id = b.deployment_id
    AND caller.app_id = b.app_id
    AND caller.workspace_id = b.workspace_id AND caller.project_id = b.project_id
    AND caller.environment_id = b.environment_id
STRAIGHT_JOIN environments caller_env ON caller_env.id = caller.environment_id
    AND caller_env.app_id = caller.app_id
STRAIGHT_JOIN deployment_connection_app_targets t
    ON t.deployment_id = b.deployment_id AND t.connection_id = b.connection_id
STRAIGHT_JOIN apps caller_app ON caller_app.id = b.app_id
    AND caller_app.workspace_id = b.workspace_id AND caller_app.project_id = b.project_id
STRAIGHT_JOIN apps target_app ON target_app.id = b.resource_id
    AND target_app.workspace_id = b.workspace_id AND target_app.project_id = b.project_id
STRAIGHT_JOIN workspaces w ON w.id = b.workspace_id
LEFT JOIN deployments live ON live.id = target_app.current_deployment_id
LEFT JOIN environments live_env ON live_env.id = live.environment_id
    AND live_env.app_id = live.app_id
WHERE b.deployment_id IN (sqlc.slice(caller_deployment_ids))
    AND (
        b.deployment_id > sqlc.arg(after_caller_deployment_id)
        OR (b.deployment_id = sqlc.arg(after_caller_deployment_id) AND b.connection_id > sqlc.arg(after_connection_id))
    )
    AND b.resource_type = 'app'
    AND b.resource_id <> b.app_id
    AND b.name NOT LIKE 'unkey%'
    AND b.name <> 'self'
ORDER BY b.deployment_id, b.connection_id
LIMIT ?;

-- name: FindPrivateNetworkPreviewDeploymentId :one
-- FindPrivateNetworkPreviewDeploymentId returns the newest deployment that was
-- ever ready on a git branch and fork in a preview environment of the app. It
-- reads app_branch_created_idx newest first and stops at the first match.
SELECT candidate.id
FROM deployments candidate
INNER JOIN environments candidate_env ON candidate_env.id = candidate.environment_id
    AND candidate_env.app_id = candidate.app_id
WHERE candidate.app_id = sqlc.arg(app_id)
    AND candidate.workspace_id = sqlc.arg(workspace_id)
    AND candidate.project_id = sqlc.arg(project_id)
    AND candidate.git_branch = sqlc.arg(git_branch)
    AND COALESCE(candidate.fork_repository_full_name, '') = sqlc.arg(fork_repository_full_name)
    AND candidate.source = 'git'
    AND candidate_env.kind = 'preview'
    AND candidate.first_ready_at IS NOT NULL
ORDER BY candidate.created_at DESC, candidate.id DESC
LIMIT 1;

-- name: FindPrivateNetworkEnvironmentDeploymentId :one
-- FindPrivateNetworkEnvironmentDeploymentId returns the newest deployment that
-- was ever ready in a preview environment of the app. It reads
-- app_environment_created_idx newest first and stops at the first match. The
-- hint keeps MySQL from sorting a small environment through environment_idx.
SELECT candidate.id
FROM deployments candidate FORCE INDEX (app_environment_created_idx)
INNER JOIN environments target_env ON target_env.id = candidate.environment_id
    AND target_env.app_id = candidate.app_id
WHERE candidate.app_id = sqlc.arg(app_id)
    AND candidate.environment_id = sqlc.arg(environment_id)
    AND candidate.workspace_id = sqlc.arg(workspace_id)
    AND candidate.project_id = sqlc.arg(project_id)
    AND target_env.kind = 'preview'
    AND candidate.first_ready_at IS NOT NULL
ORDER BY candidate.created_at DESC, candidate.id DESC
LIMIT 1;

-- name: ListPrivateNetworkTargets :many
-- ListPrivateNetworkTargets returns the given deployments that are ready, meant
-- to run, and running in a region on the platform.
SELECT
    target.id,
    target.workspace_id,
    target.project_id,
    target.app_id,
    target.environment_id,
    target.port
FROM deployments target
WHERE target.id IN (sqlc.slice(deployment_ids))
    AND target.status = 'ready'
    AND target.desired_state = 'running'
    AND EXISTS (
        SELECT 1 FROM deployment_topology target_dt
        INNER JOIN regions target_region ON target_region.id = target_dt.region_id
        WHERE target_dt.deployment_id = target.id
            AND target_dt.desired_status = 'running'
            AND target_region.platform = sqlc.arg(platform)
    );
