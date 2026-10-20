-- name: LockRowsOfNames :many
-- LockRowsOfNames locks every row of the names, and the gaps between them, so
-- no writer can add or change a row of these names until the transaction ends.
SELECT name, expires_at_ns
FROM ratelimit_gcra
WHERE name IN (sqlc.slice(names))
FOR UPDATE;
