-- name: FlipSentinelDeployStatusIfProgressing :execrows
-- FlipSentinelDeployStatusIfProgressing flips deploy_status from progressing
-- to the target status, guarding against concurrent writers (e.g. the Deploy
-- worker marking failed on timeout) by only updating rows whose current
-- status is still 'progressing'. Returns the number of rows affected; the
-- caller should treat 0 as "someone else already moved this sentinel out of
-- progressing" and skip follow-up side effects (NotifyReady, etc.).
UPDATE sentinels SET
  deploy_status = sqlc.arg(deploy_status),
  updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
  AND deploy_status = 'progressing';
