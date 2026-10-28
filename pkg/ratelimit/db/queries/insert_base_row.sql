-- name: InsertBaseRow :exec
-- InsertBaseRow inserts the row that cleanup merges closed buckets into. Its
-- bucket_start_ns is 0 and its pod_id is empty. Cleanup deletes the previous
-- base row in the same transaction.
INSERT INTO ratelimit_state (
    fingerprint,
    bucket_start_ns,
    pod_id,
    consumed_ns,
    reset_at_ns,
    compact_after_ns
) VALUES (
    sqlc.arg(fingerprint),
    0,
    '',
    sqlc.arg(consumed_ns),
    sqlc.arg(reset_at_ns),
    sqlc.arg(compact_after_ns)
);
