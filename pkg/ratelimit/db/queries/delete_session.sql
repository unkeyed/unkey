-- name: DeleteSession :execrows
-- DeleteSession deletes one session row, but only if consumed_ns still has the
-- value the caller read. A late write that raised it in the meantime keeps the
-- row, so cleanup never deletes a value it did not check.
DELETE FROM ratelimit_state
WHERE fingerprint = sqlc.arg(fingerprint)
    AND session_id = sqlc.arg(session_id)
    AND consumed_ns = sqlc.arg(consumed_ns)
LIMIT 1;
