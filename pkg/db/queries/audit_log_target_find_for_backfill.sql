-- name: FindAuditLogTargetsForBackfill :many
-- FindAuditLogTargetsForBackfill fetches every target row attached to a set
-- of audit_log_ids in one query. Called by the backfill VO after
-- FindAuditLogsForBackfill so we go from "page of N parents" to "page of N
-- parents with all their targets" in two MySQL reads, never N+1.
--
-- The unique index on (audit_log_id, id) covers the IN-list lookup. Order
-- by audit_log_id, pk so the VO's grouping pass sees a deterministic
-- per-event target sequence.
SELECT audit_log_id, type, id, name, meta
FROM audit_log_target
WHERE audit_log_id IN (sqlc.slice('audit_log_ids'))
ORDER BY audit_log_id, pk;
