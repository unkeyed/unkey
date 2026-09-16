-- name: FindAppSourceOciByAppId :one
SELECT
    pk,
    workspace_id,
    app_id,
    image_reference,
    created_at,
    updated_at
FROM app_source_oci
WHERE app_id = sqlc.arg(app_id);
