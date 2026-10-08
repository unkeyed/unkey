-- name: CountUnkeyRootKeysByWorkspace :one
-- CountUnkeyRootKeysByWorkspace counts every unkey_root_keys row in a workspace. Intended
-- for tests that prove a failed call wrote nothing.
SELECT COUNT(*)
FROM unkey_root_keys
WHERE workspace_id = sqlc.arg(workspace_id);
