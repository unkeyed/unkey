-- name: MarkSentinelFailedIfProgressing :exec
-- Conditional flip to `failed`, only when the sentinel is currently
-- `progressing`. Used by Deploy's compensation on abnormal exit so a
-- concurrent ReportSentinelStatus that already flipped the sentinel to
-- `ready` (the authoritative convergence path) doesn't get overwritten
-- with `failed`.
UPDATE sentinels
SET deploy_status = 'failed',
    updated_at    = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
  AND deploy_status = 'progressing';
