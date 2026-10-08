-- name: UpdateLogdrainsMax :exec
-- UpdateLogdrainsMax sets how many log drains a workspace may configure.
-- Intended for tests: UpsertLimit leaves logdrains_max at its default.
UPDATE `limits`
SET logdrains_max = sqlc.arg(logdrains_max)
WHERE workspace_id = sqlc.arg(workspace_id);
