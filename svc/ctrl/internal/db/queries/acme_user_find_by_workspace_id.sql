-- name: FindAcmeUserByWorkspaceID :one
SELECT * FROM acme_users WHERE workspace_id = ? LIMIT 1;
