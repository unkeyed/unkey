-- name: ListRetainedAppConnections :many
SELECT g.id, g.target_app_id, a.name AS target_app_name, g.deployments
FROM (
    SELECT b.connection_id AS id, b.resource_id AS target_app_id, COUNT(DISTINCT b.deployment_id) AS deployments
    FROM (
        SELECT DISTINCT k.connection_id
        FROM deployment_connections k FORCE INDEX (deployment_connections_app_env_connection_idx)
        WHERE k.app_id = sqlc.arg(app_id)
            AND k.environment_id = sqlc.arg(environment_id)
            AND k.connection_id >= sqlc.arg(id_cursor)
    ) AS ids
    LEFT JOIN app_connections current_rule ON current_rule.id = ids.connection_id
    STRAIGHT_JOIN deployment_connections b FORCE INDEX (deployment_connections_app_env_connection_idx)
        ON b.app_id = sqlc.arg(app_id)
        AND b.environment_id = sqlc.arg(environment_id)
        AND b.connection_id = ids.connection_id
    STRAIGHT_JOIN deployments d ON d.id = b.deployment_id
        AND d.workspace_id = sqlc.arg(workspace_id)
        AND d.app_id = sqlc.arg(app_id)
        AND d.environment_id = sqlc.arg(environment_id)
    WHERE current_rule.id IS NULL
        AND b.workspace_id = sqlc.arg(workspace_id)
        AND b.project_id = sqlc.arg(project_id)
        AND b.resource_type = 'app'
    GROUP BY b.connection_id, b.resource_id
) AS g
INNER JOIN apps a ON a.id = g.target_app_id
    AND a.workspace_id = sqlc.arg(workspace_id)
    AND a.project_id = sqlc.arg(project_id)
ORDER BY g.id ASC
LIMIT ?;
