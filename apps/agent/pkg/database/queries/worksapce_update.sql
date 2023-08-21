-- name: UpdateWorkspace :exec
UPDATE
    `workspaces`
SET
    name = sqlc.arg("name"),
    slug = sqlc.arg("slug"),
    tenant_id = sqlc.arg("tenant_id"),
    plan = sqlc.arg("plan")
WHERE
    id = sqlc.arg("id")