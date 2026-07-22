-- name: ListPrunableDeployments :many
-- ListPrunableDeployments returns a bounded batch of deployments that are
-- safe to hard-delete: they reached a terminal status that can never serve
-- traffic again and last changed before the retention cutoff.
--
-- COALESCE falls back to created_at so rows whose updated_at was never
-- stamped still age out instead of surviving forever.
--
-- The NOT EXISTS guard is a second safety boundary against deleting a row an
-- app still references. Stopped deployments are excluded by the caller because
-- preview deployments remain wakeable regardless of this pointer.
--
-- LIMIT bounds each batch; the cron loops until a batch returns fewer rows
-- than the limit.
SELECT d.id
FROM deployments d
WHERE d.status IN ('failed', 'cancelled', 'superseded', 'skipped')
  AND COALESCE(d.updated_at, d.created_at) < sqlc.arg('cutoff')
  AND NOT EXISTS (
    SELECT 1 FROM apps a WHERE a.current_deployment_id = d.id
  )
ORDER BY d.pk
LIMIT ?;
