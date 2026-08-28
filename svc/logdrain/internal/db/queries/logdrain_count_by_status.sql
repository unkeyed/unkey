-- CountLogdrainsByStatus provides low-cardinality service health metrics.
-- Callers explicitly zero absent (status, stream) pairs to avoid retaining
-- stale gauges.
-- name: CountLogdrainsByStatus :many
SELECT
  CASE
    WHEN d.enabled = false THEN 'disabled'
    WHEN d.status = 'paused_by_failure' THEN 'paused_by_failure'
    ELSE 'enabled'
  END AS status,
  d.stream,
  COUNT(*) AS drains
FROM logdrains d
GROUP BY
  CASE
    WHEN d.enabled = false THEN 'disabled'
    WHEN d.status = 'paused_by_failure' THEN 'paused_by_failure'
    ELSE 'enabled'
  END,
  d.stream;
