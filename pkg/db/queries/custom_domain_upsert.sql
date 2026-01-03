-- name: UpsertCustomDomain :exec
INSERT INTO custom_domains (id, workspace_id, domain, challenge_type, created_at)
VALUES (?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    workspace_id = VALUES(workspace_id),
    challenge_type = VALUES(challenge_type),
    updated_at = ?;
