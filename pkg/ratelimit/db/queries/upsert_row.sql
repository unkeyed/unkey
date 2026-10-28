-- name: UpsertRow :exec
-- UpsertRow writes a pod's scaled sum for one period. The sum and its
-- delete_after_ms only grow, so GREATEST keeps a retried or reordered write
-- from undoing a newer one. A write that lands after cleanup deleted the row
-- inserts the pod's whole sum again, which is what the row held.
INSERT INTO ratelimit_state (
    fingerprint,
    period_start_ms,
    pod_id,
    scaled_sum,
    delete_after_ms
) VALUES (
    sqlc.arg(fingerprint),
    sqlc.arg(period_start_ms),
    sqlc.arg(pod_id),
    sqlc.arg(scaled_sum),
    sqlc.arg(delete_after_ms)
)
ON DUPLICATE KEY UPDATE
    scaled_sum = GREATEST(scaled_sum, VALUES(scaled_sum)),
    delete_after_ms = GREATEST(delete_after_ms, VALUES(delete_after_ms));
