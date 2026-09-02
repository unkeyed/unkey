-- name: FindCertificateByHostname :one
SELECT certificates.pk, certificates.id, certificates.workspace_id, certificates.hostname, certificates.certificate, certificates.encrypted_private_key, certificates.created_at, certificates.updated_at FROM certificates WHERE hostname = ?;
