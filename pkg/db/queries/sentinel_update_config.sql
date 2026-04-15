-- name: UpdateSentinelConfig :exec
-- UpdateSentinelConfig updates a sentinel's configuration and deploy status.
-- Used by SentinelService.Deploy() to apply new config before triggering krane.
UPDATE sentinels SET
  image = sqlc.arg(image),
  cpu_millicores = sqlc.arg(cpu_millicores),
  memory_mib = sqlc.arg(memory_mib),
  desired_replicas = sqlc.arg(desired_replicas),
  deploy_status = sqlc.arg(deploy_status),
  updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);
