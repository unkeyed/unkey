-- name: ListSessions :many
-- ListSessions returns the sessions of a fingerprint, newest first. Callers
-- compute reset_at from them and merge their own sessions by session_id,
-- keeping the larger consumed_ns. If the result reaches the LIMIT, older
-- sessions may be missing, and the caller must not use the result for a
-- decision: a missing session can open the limit too early.
SELECT session_id, anchor_at_ns, consumed_ns, compact_after_ms
FROM ratelimit_state
WHERE fingerprint = sqlc.arg(fingerprint)
ORDER BY anchor_at_ns DESC, session_id
LIMIT ?;
