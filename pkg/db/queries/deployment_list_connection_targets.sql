-- name: ListAppConnectionTargetDeployments :many
-- The plain created_at bound lets the index seek; MySQL cannot range-scan the
-- row comparison.
SELECT d.id, d.environment_id, d.status, d.git_branch, d.image_resolved
FROM deployments d FORCE INDEX (app_status_desired_created_idx)
WHERE d.workspace_id = sqlc.arg(workspace_id) AND d.app_id = sqlc.arg(app_id)
    AND d.status = 'ready' AND d.desired_state = 'running' AND d.first_ready_at IS NOT NULL
    AND (
        sqlc.arg(cursor_id) = '' OR (
            d.created_at <= (
                SELECT c.created_at FROM deployments c
                WHERE c.id = sqlc.arg(cursor_id) AND c.workspace_id = sqlc.arg(workspace_id)
                    AND c.app_id = sqlc.arg(app_id)
            )
            AND (d.created_at, d.pk) <= (
                SELECT c.created_at, c.pk FROM deployments c
                WHERE c.id = sqlc.arg(cursor_id) AND c.workspace_id = sqlc.arg(workspace_id)
                    AND c.app_id = sqlc.arg(app_id)
            )
        )
    )
ORDER BY d.created_at DESC, d.pk DESC
LIMIT ?;
