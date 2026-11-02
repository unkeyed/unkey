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
