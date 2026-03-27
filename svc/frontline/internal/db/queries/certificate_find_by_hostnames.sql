-- name: FindBestCertificateByCandidates :one
-- FindBestCertificateByCandidates returns a single certificate row for the
-- provided candidates, preferring an exact hostname over wildcard matches.
-- MySQL does not preserve IN-list order, so exact-first behavior is enforced
-- by ORDER BY against the exact_hostname argument rather than by candidate
-- position.
--
-- For example, if candidates are ['api.example.com', '*.example.com'] and
-- exact_hostname is 'api.example.com', the query returns 'api.example.com'
-- when both rows exist.
--
-- If only '*.example.com' exists, the query returns that wildcard row as the
-- fallback.
--
-- LIMIT 1 keeps the lookup to one selected row and avoids returning unused
-- certificate and key payloads.
SELECT
  hostname,
  workspace_id,
  certificate,
  encrypted_private_key
FROM certificates
WHERE hostname IN (sqlc.slice('hostnames'))
ORDER BY hostname = sqlc.arg(exact_hostname) DESC
LIMIT 1;
