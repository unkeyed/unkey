-- name: FindAcmeUserByWorkspaceID :one
SELECT acme_users.pk, acme_users.id, acme_users.workspace_id, acme_users.encrypted_key, acme_users.registration_uri, acme_users.created_at, acme_users.updated_at FROM acme_users WHERE workspace_id = ? LIMIT 1;
