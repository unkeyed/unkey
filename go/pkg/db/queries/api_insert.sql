-- name: InsertApi :exec
INSERT INTO apis (
    id,
    name,
    workspace_id,
    auth_type,
    key_auth_id,
    created_at_m,
    deleted_at_m
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    NULL
);
