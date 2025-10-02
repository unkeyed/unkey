-- name: UpsertAnalyticsConfig :exec
INSERT INTO `analytics_config` (workspace_id, enabled, storage, config, created_at)
VALUES (?, TRUE, ?, ?, UNIX_TIMESTAMP() * 1000)
ON DUPLICATE KEY UPDATE
	enabled = TRUE,
	storage = VALUES(storage),
	config = VALUES(config),
	updated_at = UNIX_TIMESTAMP() * 1000;