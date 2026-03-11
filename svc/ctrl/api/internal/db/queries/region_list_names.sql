-- name: ListRegionNames :many
-- ListRegionNames returns the canonical region name for every region known to
-- ctrl. Certificate bootstrap uses these names to build wildcard hostnames
-- like "*.{region}.{regionalDomain}".
--
-- Result ordering is intentionally unspecified because the bootstrap flow is
-- idempotent and does not require deterministic processing order.
SELECT
  name
FROM regions;
