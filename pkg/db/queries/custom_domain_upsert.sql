-- name: UpsertCustomDomain :exec
INSERT INTO custom_domains (
    id, workspace_id, project_id, environment_id, domain,
    challenge_type, verification_status, target_cname, created_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    workspace_id = VALUES(workspace_id),
    project_id = VALUES(project_id),
    environment_id = VALUES(environment_id),
    challenge_type = VALUES(challenge_type),
    verification_status = VALUES(verification_status),
    target_cname = VALUES(target_cname),
    updated_at = ?;
