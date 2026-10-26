-- name: InsertDeploymentConnection :exec
-- Used by tests to seed saved deployment connections.
INSERT INTO deployment_connections (
    deployment_id, connection_id, workspace_id, project_id, app_id, environment_id,
    resource_type, resource_id, name, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
