-- name: LockRowsOfNames :many
-- LockRowsOfNames locks the rows of the names, and the gaps between them, so no
-- writer can add or change a row of a locked name until the transaction ends.
-- The LIMIT bounds the locked rows; rows come in name order, so the caller can
-- tell which names it saw completely.
SELECT name, expires_at_ns
FROM ratelimit_gcra
WHERE name IN (sqlc.slice(names))
ORDER BY name, writer_id
LIMIT ?
FOR UPDATE;
