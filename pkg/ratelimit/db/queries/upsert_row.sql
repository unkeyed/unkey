-- name: UpsertRow :exec
-- UpsertRow writes a pod's running totals for one bucket. Both totals only
-- grow, so GREATEST keeps a retried or reordered write from undoing a newer
-- one.
--
-- MySQL refuses the write once compact_after_ns has passed by its own clock,
-- because cleanup may already have merged the bucket: the consumed_ns value
-- becomes 0, so an update keeps the row as it is, and an insert stores a row
-- that readers skip and cleanup deletes. write_until_ns must equal
-- compact_after_ns.
INSERT INTO ratelimit_state (
    fingerprint,
    bucket_start_ns,
    pod_id,
    consumed_ns,
    reset_at_ns,
    compact_after_ns
) VALUES (
    sqlc.arg(fingerprint),
    sqlc.arg(bucket_start_ns),
    sqlc.arg(pod_id),
    IF(sqlc.arg(write_until_ns) < CAST(UNIX_TIMESTAMP(NOW(6)) * 1000000000 AS SIGNED), 0, sqlc.arg(consumed_ns)),
    sqlc.arg(reset_at_ns),
    sqlc.arg(compact_after_ns)
)
ON DUPLICATE KEY UPDATE
    reset_at_ns = IF(VALUES(consumed_ns) = 0, reset_at_ns, GREATEST(reset_at_ns, VALUES(reset_at_ns))),
    consumed_ns = GREATEST(consumed_ns, VALUES(consumed_ns));
