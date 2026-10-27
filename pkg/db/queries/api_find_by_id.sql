-- name: FindApiByID :one
SELECT apis.pk, apis.id, apis.name, apis.workspace_id, apis.project_id, apis.ip_whitelist, apis.auth_type, apis.key_auth_id, apis.created_at_m, apis.updated_at_m, apis.deleted_at_m, apis.delete_protection FROM apis WHERE id = ?;

-- name: FindApiByIDForUpdate :one
-- FindApiByIDForUpdate locks the API so mutation checks remain valid until commit.
SELECT apis.pk, apis.id, apis.name, apis.workspace_id, apis.project_id, apis.ip_whitelist, apis.auth_type, apis.key_auth_id, apis.created_at_m, apis.updated_at_m, apis.deleted_at_m, apis.delete_protection FROM apis WHERE id = ? FOR UPDATE;
