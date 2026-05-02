-- name: GetLogDrainCursor :one
-- Returns one drain's cursor row for a specific group. Cursors are keyed
-- by (drain_id, group_key) because a single drain belongs to N groups
-- (one per source) and each source has its own (time_ms, last_id)
-- timeline; a drain_id-only PK would let one source's tail overshoot
-- another's and stall the slower source's processGroup at "fetch
-- returns 0 rows" forever. Reads go to the primary so the same tick
-- observes its own previous write — replication lag against RO would
-- otherwise make the optimistic-lock UPDATE ambiguous between "another
-- replica won" and "your last write hasn't replicated yet".
SELECT drain_id, group_key, time_ms, last_id, blocked, blocked_reason, updated_at
FROM log_drain_cursors
WHERE drain_id = sqlc.arg(drain_id)
  AND group_key = sqlc.arg(group_key);
