-- name: ListWorkspaceBuildConcurrencyAbove :many
SELECT workspace_id, builds_concurrent_max
FROM `limits`
WHERE builds_concurrent_max > sqlc.arg(builds_concurrent_max)
ORDER BY workspace_id;
