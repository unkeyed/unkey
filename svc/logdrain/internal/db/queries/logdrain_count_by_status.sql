-- CountLogdrainsByStatus provides low-cardinality service health metrics.
-- Callers explicitly zero absent (status, stream) pairs to avoid retaining
-- stale gauges.
-- name: CountLogdrainsByStatus :many
SELECT
  CASE
    WHEN d.enabled = false THEN 'disabled'
    WHEN s.status = 'paused_by_failure' THEN 'paused_by_failure'
    ELSE 'enabled'
  END AS status,
  d.stream,
  COUNT(*) AS drains
FROM logdrains d
JOIN logdrain_state s ON s.logdrain_id = d.id
GROUP BY
  CASE
    WHEN d.enabled = false THEN 'disabled'
    WHEN s.status = 'paused_by_failure' THEN 'paused_by_failure'
    ELSE 'enabled'
  END,
  d.stream;
