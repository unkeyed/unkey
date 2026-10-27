-- name: InsertNotificationDestination :exec
INSERT INTO notification_destinations (id, workspace_id, name, channel, config, enabled, created_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(name), sqlc.arg(channel), sqlc.arg(config), sqlc.arg(enabled), sqlc.arg(created_at));
