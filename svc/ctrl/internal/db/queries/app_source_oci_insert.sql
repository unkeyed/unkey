-- name: InsertAppSourceOci :exec
INSERT INTO app_source_oci (
    workspace_id,
    app_id,
    image_reference,
    created_at,
    updated_at
) VALUES (
    sqlc.arg(workspace_id),
    sqlc.arg(app_id),
    sqlc.arg(image_reference),
    sqlc.arg(created_at),
    sqlc.arg(updated_at)
);
