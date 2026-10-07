-- name: DeleteWorkspaceFlagOverride :exec
-- DeleteWorkspaceFlagOverride restores inheritance for this workspace only.
-- Deleting an absent override is intentionally idempotent.
DELETE FROM workspace_flag_overrides
WHERE workspace_id = sqlc.arg(workspace_id) AND flag_id = sqlc.arg(flag_id);
