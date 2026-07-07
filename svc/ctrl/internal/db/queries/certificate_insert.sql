-- name: InsertCertificate :exec
INSERT INTO certificates (id, workspace_id, hostname, certificate, encrypted_private_key, created_at)
VALUES (?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE
workspace_id = VALUES(workspace_id),
hostname = VALUES(hostname),
certificate = VALUES(certificate),
encrypted_private_key = VALUES(encrypted_private_key),
updated_at = ?;
