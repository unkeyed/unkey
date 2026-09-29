-- name: DeleteUnkeyPermissionsByPrincipal :exec
-- DeleteUnkeyPermissionsByPrincipal removes principal permissions before a replacement.
DELETE FROM unkey_principal_permissions
WHERE for_workspace_id = sqlc.arg(for_workspace_id)
    AND principal_type = sqlc.arg(principal_type)
    AND principal_id = sqlc.arg(principal_id);
