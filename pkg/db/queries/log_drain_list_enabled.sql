-- name: ListEnabledLogDrains :many
SELECT
    d.id,
    d.workspace_id,
    d.project_id,
    d.name,
    d.provider,
    d.config,
    d.sources,
    d.environments,
    d.apps,
    d.filters,
    d.delivery_mode,
    c.source AS credential_source,
    c.encrypted_credentials,
    c.encryption_key_id,
    c.oauth_grant_id,
    s.consecutive_failures,
    s.paused_reason
FROM log_drains d
LEFT JOIN log_drain_credentials c ON c.drain_id = d.id
LEFT JOIN log_drain_state s ON s.drain_id = d.id
WHERE d.deleted_at IS NULL
  AND d.enabled = true
  AND (s.paused_reason IS NULL OR s.paused_reason = '');
