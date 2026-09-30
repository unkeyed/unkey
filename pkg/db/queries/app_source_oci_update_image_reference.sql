-- name: UpdateAppSourceOciImageReference :exec
UPDATE app_source_oci
SET image_reference = sqlc.arg(image_reference),
    updated_at = sqlc.arg(updated_at)
WHERE app_id = sqlc.arg(app_id)
  AND workspace_id = sqlc.arg(workspace_id);
