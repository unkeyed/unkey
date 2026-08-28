-- CountLogdrainsByStatus provides low-cardinality service health metrics.
-- Callers explicitly zero absent (status, stream) pairs to avoid retaining
-- stale gauges.
-- name: CountLogdrainsByStatus :many
SELECT
  d.status,
  d.stream,
  COUNT(*) AS drains
FROM logdrains d
GROUP BY d.status, d.stream;
