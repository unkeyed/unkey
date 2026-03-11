-- name: UpsertCustomDomain :exec
-- UpsertCustomDomain creates or refreshes the platform-managed custom-domain
-- record used by certificate bootstrap.
--
-- On duplicate domain, this query updates mutable routing and verification
-- fields while preserving immutable identity and creation timestamp.
--
-- Example: if "*.us-west-2.example.com" already exists, rerunning bootstrap
-- keeps the same row identity and only refreshes fields included in the
-- ON DUPLICATE KEY UPDATE clause plus updated_at.
INSERT INTO custom_domains (
  id,
  workspace_id,
  project_id,
  app_id,
  environment_id,
  domain,
  challenge_type,
  verification_status,
  verification_token,
  target_cname,
  created_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
  workspace_id = VALUES(workspace_id),
  project_id = VALUES(project_id),
  app_id = VALUES(app_id),
  environment_id = VALUES(environment_id),
  challenge_type = VALUES(challenge_type),
  verification_status = VALUES(verification_status),
  target_cname = VALUES(target_cname),
  updated_at = ?;
