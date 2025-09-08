-- name: ListByWorkspace :many
SELECT * FROM kv
WHERE workspace_id = ? 
AND (ttl IS NULL OR ttl > ?)
AND id > ?
ORDER BY id ASC
LIMIT ?;