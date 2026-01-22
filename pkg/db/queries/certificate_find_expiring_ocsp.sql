-- name: FindCertificatesWithExpiringOCSP :many
SELECT pk, id, workspace_id, hostname, certificate, ocsp_staple, ocsp_expires_at
FROM certificates
WHERE ocsp_expires_at IS NULL OR ocsp_expires_at < ?
LIMIT 1000;
