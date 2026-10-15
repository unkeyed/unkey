-- name: ListPrivateNetworkConnections :many
-- ListPrivateNetworkConnections returns the directed app connections saved by
-- the given caller deployments, with the target deployment each connection
-- selects now. The target is empty unless that deployment is ready on the
-- platform. Names starting with unkey and the caller app's own slug are
-- reserved. Callers must come from ListPrivateNetworkReplicas in the same
-- transaction, which decides which deployments are active callers.
-- Rows are paged by (caller deployment, connection) in the order of
-- deployment_connections' unique (deployment_id, connection_id) index, so the
-- LIMIT bounds both the rows read and the rows returned.
WITH connection_candidates AS (
    SELECT
        b.connection_id,
        b.name AS connection_name,
        b.workspace_id,
        b.project_id,
        b.resource_id AS target_app_id,
        target_app.slug AS target_app_slug,
        w.k8s_namespace,
        caller.id AS caller_deployment_id,
        CASE
            WHEN t.selection_mode = 'deployment' THEN t.target_deployment_id
            WHEN t.selection_mode = 'automatic'
                AND caller_env.kind = 'production'
                THEN target_app.current_deployment_id
            WHEN t.selection_mode = 'automatic'
                AND caller.source = 'git'
                AND COALESCE(caller.git_branch, '') <> ''
                THEN (
                SELECT candidate.id
                FROM deployments candidate
                INNER JOIN environments candidate_env ON candidate_env.id = candidate.environment_id
                    AND candidate_env.app_id = candidate.app_id
                WHERE candidate.app_id = b.resource_id
                    AND candidate.workspace_id = b.workspace_id
                    AND candidate.project_id = b.project_id
                    AND candidate_env.kind = 'preview'
                    AND candidate.source = 'git'
                    AND candidate.git_branch = caller.git_branch
                    AND COALESCE(candidate.fork_repository_full_name, '') = COALESCE(caller.fork_repository_full_name, '')
                    AND candidate.first_ready_at IS NOT NULL
                ORDER BY candidate.created_at DESC, candidate.id DESC
                LIMIT 1
            )
            WHEN t.selection_mode = 'environment' THEN COALESCE(
                (
                    SELECT live.id
                    FROM deployments live
                    INNER JOIN environments live_env ON live_env.id = live.environment_id
                        AND live_env.app_id = live.app_id
                    WHERE live.id = target_app.current_deployment_id
                        AND live.environment_id = t.target_environment_id
                ),
                (
                    SELECT candidate.id
                    FROM deployments candidate
                    WHERE candidate.app_id = b.resource_id
                        AND candidate.environment_id = t.target_environment_id
                        AND candidate.workspace_id = b.workspace_id
                        AND candidate.project_id = b.project_id
                        AND candidate.first_ready_at IS NOT NULL
                        AND EXISTS (
                            SELECT 1 FROM environments target_env
                            WHERE target_env.id = t.target_environment_id
                                AND target_env.app_id = b.resource_id
                                AND target_env.kind = 'preview'
                        )
                    ORDER BY candidate.created_at DESC, candidate.id DESC
                    LIMIT 1
                )
            )
        END AS selected_deployment_id
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
    WHERE b.deployment_id IN (sqlc.slice(caller_deployment_ids))
        AND (
            b.deployment_id > sqlc.arg(after_caller_deployment_id)
            OR (b.deployment_id = sqlc.arg(after_caller_deployment_id) AND b.connection_id > sqlc.arg(after_connection_id))
        )
        AND b.resource_type = 'app'
        AND b.resource_id <> b.app_id
        AND b.name NOT LIKE 'unkey%'
        AND b.name <> caller_app.slug COLLATE utf8mb4_0900_as_cs
    ORDER BY b.deployment_id, b.connection_id
    LIMIT ?
)
SELECT
    c.workspace_id,
    c.project_id,
    c.target_app_id AS app_id,
    c.target_app_slug AS app_slug,
    c.k8s_namespace,
    COALESCE(target.id, '') AS deployment_id,
    COALESCE(target.port, 0) AS port,
    COALESCE(target.environment_id, '') AS environment_id,
    c.caller_deployment_id,
    c.connection_id,
    c.connection_name
FROM connection_candidates c
LEFT JOIN deployments target ON target.id = c.selected_deployment_id
    AND target.app_id = c.target_app_id
    AND target.workspace_id = c.workspace_id AND target.project_id = c.project_id
    AND target.status = 'ready'
    AND target.desired_state = 'running'
    AND EXISTS (
        SELECT 1 FROM deployment_topology target_dt
        INNER JOIN regions target_region ON target_region.id = target_dt.region_id
        WHERE target_dt.deployment_id = target.id
            AND target_dt.desired_status = 'running'
            AND target_region.platform = sqlc.arg(platform)
    )
ORDER BY c.caller_deployment_id, c.connection_id;
