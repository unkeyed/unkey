-- name: ListAppBindingsByApp :many
SELECT id, name
FROM app_bindings
WHERE workspace_id = sqlc.arg(workspace_id)
    AND project_id = sqlc.arg(project_id)
    AND app_id = sqlc.arg(app_id)
    AND environment_id = sqlc.arg(environment_id)
    AND resource_type = 'app'
    AND resource_id <> app_id
ORDER BY pk;
