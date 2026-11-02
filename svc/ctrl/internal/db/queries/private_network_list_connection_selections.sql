-- name: ListPrivateNetworkConnectionSelections :many
-- ListPrivateNetworkConnectionSelections returns the directed app connections
-- saved by the given caller deployments, with the saved target selection and
-- the caller and live deployment facts that ListPrivateNetworkConnections
-- needs to resolve it. LiveEnvironmentID is the environment of the target
-- app's live deployment, empty when there is none. Callers must come from
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
    caller.fork_key AS caller_fork_key,
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
