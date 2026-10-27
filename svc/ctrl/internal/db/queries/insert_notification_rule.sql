-- name: InsertNotificationRule :exec
INSERT INTO notification_subscriptions (id, workspace_id, name, notification_type, project_id, app_id, environment_id, enabled, created_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(name), sqlc.arg(notification_type), sqlc.narg(project_id), sqlc.narg(app_id), sqlc.narg(environment_id), sqlc.arg(enabled), sqlc.arg(created_at));
