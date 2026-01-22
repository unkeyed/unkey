-- name: UpdateCertificateOCSP :exec
UPDATE certificates
SET ocsp_staple = ?, ocsp_expires_at = ?, updated_at = ?
WHERE hostname = ?;
