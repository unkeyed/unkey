-- name: UpdateUnkeyRootKeyLastUsedAt :exec
-- UpdateUnkeyRootKeyLastUsedAt records when a new-format root key was last
-- used. Intended for tests.
UPDATE unkey_root_keys
SET last_used_at = sqlc.arg(last_used_at)
WHERE id = sqlc.arg(id)
    AND workspace_id = sqlc.arg(workspace_id);
