-- name: DeleteWorkspaceFlagOverride :exec
-- DeleteWorkspaceFlagOverride restores inheritance for this workspace only.
-- Deleting an absent override is intentionally idempotent.
DELETE FROM flag_overrides
WHERE workspace_id = sqlc.arg(workspace_id) AND flag_slug = sqlc.arg(flag_slug);
