-- name: ListAppConnectionTargetDeployments :many
SELECT d.id, d.app_id, d.environment_id, d.status, d.git_branch, d.image_resolved, d.created_at
FROM deployments d
WHERE d.workspace_id = sqlc.arg(workspace_id) AND d.project_id = sqlc.arg(project_id)
    AND d.app_id = sqlc.arg(app_id) AND d.first_ready_at IS NOT NULL
    AND d.status IN ('ready', 'stopped')
    AND (
        sqlc.arg(cursor_id) = '' OR (
            d.created_at <= (
                SELECT c.created_at FROM deployments c
                WHERE c.id = sqlc.arg(cursor_id) AND c.workspace_id = sqlc.arg(workspace_id)
                    AND c.project_id = sqlc.arg(project_id) AND c.app_id = sqlc.arg(app_id)
            )
            AND (d.created_at, d.id) <= (
                SELECT c.created_at, c.id FROM deployments c
                WHERE c.id = sqlc.arg(cursor_id) AND c.workspace_id = sqlc.arg(workspace_id)
                    AND c.project_id = sqlc.arg(project_id) AND c.app_id = sqlc.arg(app_id)
            )
        )
    )
ORDER BY d.created_at DESC, d.id DESC
LIMIT ?;
