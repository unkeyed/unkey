-- name: FindAuditLogTargetsForLogs :many
-- FindAuditLogTargetsForLogs fetches every target row attached to the given
-- audit log IDs. Called by the export worker after FindUnexportedAuditLogs to
-- fan out (event, target) pairs into per-target ClickHouse rows. Ordered by
-- audit_log_id then pk so the worker's grouping pass sees a deterministic
-- per-event target sequence (also keeps insert blocks byte-stable for retry
-- dedup).
SELECT audit_log_id, type, id, name, meta
FROM audit_log_target
WHERE audit_log_id IN (sqlc.slice('audit_log_ids'))
ORDER BY audit_log_id, pk;
