-- name: MarkAuditLogsExported :exec
-- MarkAuditLogsExported flips the outbox flag for rows whose ClickHouse
-- insert has been confirmed. The redundant `AND exported = false` guard is
-- belt-and-braces idempotency: if two cron invocations ever raced (Restate
-- VO serialization should prevent this, but defense in depth) the second
-- update is a noop instead of double-marking.
UPDATE audit_log
SET exported = true
WHERE pk IN (sqlc.slice('pks'))
  AND exported = false;
