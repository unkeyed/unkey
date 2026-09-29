-- name: ListUnkeyPermissionsByPrincipal :many
-- ListUnkeyPermissionsByPrincipal loads permissions for exactly one principal
-- and authorized workspace. The same ID under another type or workspace is excluded.
SELECT slug FROM unkey_principal_permissions
WHERE for_workspace_id = sqlc.arg(for_workspace_id)
  AND principal_type = sqlc.arg(principal_type)
  AND principal_id = sqlc.arg(principal_id);
