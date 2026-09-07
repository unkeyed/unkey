-- name: ListDeploymentGCCandidates :many
-- Lists terminal deployments whose retention window has expired. Production
-- keeps every deployment from the time window plus its ten newest successful
-- revisions. Preview deployments age from their last lifecycle update.
SELECT d.pk, d.id
FROM deployments d
JOIN environments e ON e.id = d.environment_id
JOIN apps a ON a.id = d.app_id
WHERE d.pk > sqlc.arg(pagination_cursor)
  AND d.status IN ('failed', 'skipped', 'stopped', 'superseded', 'cancelled')
  AND (d.status != 'stopped' OR d.desired_state = 'stopped')
  AND (a.current_deployment_id IS NULL OR a.current_deployment_id != d.id)
  AND NOT EXISTS (
    SELECT 1 FROM instances i WHERE i.deployment_id = d.id
  )
  AND NOT EXISTS (
    SELECT 1
    FROM frontline_routes fr
    WHERE fr.deployment_id = d.id
      AND fr.sticky IN ('branch', 'environment', 'live')
  )
  AND (
    d.deleted_at <= sqlc.arg(recovery_cutoff)
    OR (d.deleted_at IS NULL AND (
    (
      e.kind = 'preview'
      AND GREATEST(COALESCE(d.updated_at, d.created_at), COALESCE(d.restored_at, 0)) < CAST(sqlc.arg(preview_cutoff) AS SIGNED)
    )
    OR
    (
      e.kind = 'production'
      AND GREATEST(d.created_at, COALESCE(d.restored_at, 0)) < CAST(sqlc.arg(production_cutoff) AS SIGNED)
      AND (
        d.status != 'stopped'
        OR (
          SELECT COUNT(*)
          FROM deployments newer
          WHERE newer.app_id = d.app_id
            AND newer.environment_id = d.environment_id
            AND newer.deleted_at IS NULL
            AND newer.status IN ('ready', 'stopped')
            AND (
              newer.created_at > d.created_at
              OR (newer.created_at = d.created_at AND newer.pk > d.pk)
            )
        ) >= CAST(sqlc.arg(keep_successful) AS UNSIGNED)
      )
    )
    ))
  )
ORDER BY d.pk
LIMIT ?;
