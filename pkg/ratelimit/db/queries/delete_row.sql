-- name: DeleteRow :execrows
-- DeleteRow deletes one row, but only if it still has the values the caller
-- read. Another cleanup that changed it in the meantime keeps the row, so
-- cleanup never deletes a value it did not check.
DELETE FROM ratelimit_state
WHERE fingerprint = sqlc.arg(fingerprint)
    AND bucket_start_ns = sqlc.arg(bucket_start_ns)
    AND pod_id = sqlc.arg(pod_id)
    AND consumed_ns = sqlc.arg(consumed_ns)
    AND reset_at_ns = sqlc.arg(reset_at_ns)
LIMIT 1;
