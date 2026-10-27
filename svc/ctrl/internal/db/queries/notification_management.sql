-- name: LockNotificationWorkspace :one
SELECT id FROM workspaces
WHERE id = sqlc.arg(workspace_id) AND enabled = TRUE AND deleted_at_m IS NULL
FOR UPDATE;

-- name: NotificationManagementScopeValid :one
SELECT EXISTS (
  SELECT 1 FROM workspaces w
  WHERE w.id = sqlc.arg(workspace_id) AND w.enabled = TRUE AND w.deleted_at_m IS NULL
    AND (sqlc.arg(project_id) = '' OR EXISTS (SELECT 1 FROM projects p WHERE p.id = sqlc.arg(project_id) AND p.workspace_id = w.id AND p.deleted_at IS NULL))
    AND (sqlc.arg(app_id) = '' OR EXISTS (SELECT 1 FROM apps a WHERE a.id = sqlc.arg(app_id) AND a.project_id = sqlc.arg(project_id) AND a.workspace_id = w.id AND a.deleted_at IS NULL))
    AND (sqlc.arg(environment_id) = '' OR EXISTS (SELECT 1 FROM environments e WHERE e.id = sqlc.arg(environment_id) AND e.project_id = sqlc.arg(project_id) AND e.app_id = sqlc.arg(app_id) AND e.workspace_id = w.id AND e.deleted_at IS NULL))
) AS valid;

-- name: FindNotificationDestination :one
SELECT pk, id, workspace_id, name, channel, config, enabled, created_at, updated_at
FROM notification_destinations WHERE workspace_id = sqlc.arg(workspace_id) AND id = sqlc.arg(id);

-- name: ListNotificationDestinations :many
SELECT pk, id, workspace_id, name, channel, config, enabled, created_at, updated_at
FROM notification_destinations
WHERE workspace_id = sqlc.arg(workspace_id) AND id > sqlc.arg(page_cursor)
ORDER BY id LIMIT ?;

-- name: UpdateNotificationDestination :exec
UPDATE notification_destinations SET name = sqlc.arg(name), config = sqlc.arg(config), enabled = sqlc.arg(enabled), updated_at = sqlc.arg(updated_at)
WHERE workspace_id = sqlc.arg(workspace_id) AND id = sqlc.arg(id);

-- name: NotificationDestinationReferenced :one
SELECT EXISTS (SELECT 1 FROM notification_subscription_targets WHERE workspace_id = sqlc.arg(workspace_id) AND destination_id = sqlc.arg(id)) AS referenced;

-- name: DeleteNotificationDestination :exec
DELETE FROM notification_destinations WHERE workspace_id = sqlc.arg(workspace_id) AND id = sqlc.arg(id);

-- name: FindNotificationRule :one
SELECT pk, id, workspace_id, name, notification_type, project_id, app_id, environment_id, enabled, created_at, updated_at
FROM notification_subscriptions WHERE workspace_id = sqlc.arg(workspace_id) AND id = sqlc.arg(id);

-- name: ListNotificationRules :many
SELECT pk, id, workspace_id, name, notification_type, project_id, app_id, environment_id, enabled, created_at, updated_at
FROM notification_subscriptions
WHERE workspace_id = sqlc.arg(workspace_id) AND id > sqlc.arg(page_cursor)
ORDER BY id LIMIT ?;

-- name: UpdateNotificationRule :exec
UPDATE notification_subscriptions
SET name = sqlc.arg(name), notification_type = sqlc.arg(notification_type), project_id = sqlc.narg(project_id), app_id = sqlc.narg(app_id), environment_id = sqlc.narg(environment_id), enabled = sqlc.arg(enabled), updated_at = sqlc.arg(updated_at)
WHERE workspace_id = sqlc.arg(workspace_id) AND id = sqlc.arg(id);

-- name: DeleteNotificationRule :exec
DELETE FROM notification_subscriptions WHERE workspace_id = sqlc.arg(workspace_id) AND id = sqlc.arg(id);

-- name: ListNotificationRuleTargets :many
SELECT pk, id, workspace_id, subscription_id, channel, audience, user_id, destination_id, created_at, updated_at
FROM notification_subscription_targets
WHERE workspace_id = sqlc.arg(workspace_id) AND subscription_id = sqlc.arg(subscription_id)
ORDER BY id LIMIT 101;

-- name: DeleteNotificationRuleTargets :exec
DELETE FROM notification_subscription_targets WHERE workspace_id = sqlc.arg(workspace_id) AND subscription_id = sqlc.arg(subscription_id);

-- name: ListNotificationUserPreferences :many
SELECT notification_type, enabled FROM notification_preferences
WHERE workspace_id = sqlc.arg(workspace_id) AND user_id = sqlc.arg(user_id) AND channel = 'email'
ORDER BY notification_type LIMIT 101;

-- name: UpsertNotificationPreference :exec
INSERT INTO notification_preferences (id, workspace_id, user_id, notification_type, channel, enabled, created_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(user_id), sqlc.arg(notification_type), 'email', sqlc.arg(enabled), sqlc.arg(created_at))
ON DUPLICATE KEY UPDATE enabled = VALUES(enabled), updated_at = VALUES(created_at);
