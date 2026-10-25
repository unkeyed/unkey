-- name: ListNotificationTargets :many
SELECT t.channel, t.audience, t.user_id, t.destination_id,
       d.config AS destination_config, d.enabled AS destination_enabled
FROM notification_subscriptions s
JOIN notification_subscription_targets t ON t.workspace_id = s.workspace_id AND t.subscription_id = s.id
LEFT JOIN notification_destinations d ON d.workspace_id = t.workspace_id AND d.id = t.destination_id AND d.channel = t.channel
WHERE s.workspace_id = sqlc.arg(workspace_id)
  AND s.notification_type = sqlc.arg(notification_type)
  AND s.enabled = TRUE
  AND (s.project_id IS NULL OR s.project_id = sqlc.arg(project_id))
  AND (s.app_id IS NULL OR s.app_id = sqlc.arg(app_id))
  AND (s.environment_id IS NULL OR s.environment_id = sqlc.arg(environment_id))
ORDER BY t.id
LIMIT 1001;

-- name: ListNotificationPreferences :many
SELECT user_id, enabled FROM notification_preferences
WHERE workspace_id = sqlc.arg(workspace_id) AND notification_type = sqlc.arg(notification_type) AND channel = 'email'
ORDER BY user_id LIMIT 10001;

-- name: FindNotification :one
SELECT pk, id, workspace_id, source_event_id, notification_type, request_hash, schema_version,
       resource_context, payload, occurred_at, expires_at, created_at, updated_at
FROM notifications WHERE workspace_id = sqlc.arg(workspace_id) AND id = sqlc.arg(id);

-- name: InsertNotificationDelivery :exec
INSERT INTO notification_deliveries (id, workspace_id, notification_id, channel, target_key, encrypted_target_snapshot, provider, status, reason, created_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(notification_id), sqlc.arg(channel), sqlc.arg(target_key), sqlc.arg(encrypted_target_snapshot), sqlc.narg(provider), sqlc.arg(status), sqlc.narg(reason), sqlc.arg(created_at));

-- name: ListNotificationDeliveries :many
SELECT pk, id, workspace_id, notification_id, channel, target_key, encrypted_target_snapshot,
       provider, provider_message_id, status, reason, attempt_count, first_attempt_at, next_attempt_at, created_at, updated_at
FROM notification_deliveries
WHERE workspace_id = sqlc.arg(workspace_id) AND notification_id = sqlc.arg(notification_id)
ORDER BY id LIMIT 1001;

-- name: ListPendingNotificationDeliveries :many
SELECT id, workspace_id FROM notification_deliveries
WHERE workspace_id = sqlc.arg(workspace_id) AND notification_id = sqlc.arg(notification_id)
  AND status IN ('pending', 'sending')
ORDER BY id LIMIT 1001;

-- name: FindNotificationDelivery :one
SELECT pk, id, workspace_id, notification_id, channel, target_key, encrypted_target_snapshot,
       provider, provider_message_id, status, reason, attempt_count, first_attempt_at, next_attempt_at, created_at, updated_at
FROM notification_deliveries WHERE workspace_id = sqlc.arg(workspace_id) AND id = sqlc.arg(id);

-- name: StartNotificationDeliveryAttempt :execrows
UPDATE notification_deliveries
SET status = 'sending', attempt_count = attempt_count + 1,
    first_attempt_at = COALESCE(first_attempt_at, sqlc.arg(now)), next_attempt_at = NULL, updated_at = sqlc.arg(now)
WHERE workspace_id = sqlc.arg(workspace_id) AND id = sqlc.arg(id)
  AND status = sqlc.arg(expected_status) AND attempt_count = sqlc.arg(expected_attempt_count)
  AND status IN ('pending', 'sending');

-- name: UpdateNotificationDelivery :execrows
UPDATE notification_deliveries
SET status = sqlc.arg(status), reason = sqlc.narg(reason), provider_message_id = sqlc.narg(provider_message_id),
    next_attempt_at = sqlc.narg(next_attempt_at), updated_at = sqlc.arg(now)
WHERE workspace_id = sqlc.arg(workspace_id) AND id = sqlc.arg(id)
  AND status = sqlc.arg(expected_status) AND attempt_count = sqlc.arg(expected_attempt_count)
  AND status IN ('pending', 'sending');

-- name: NotificationResourceBelongsToWorkspace :one
SELECT EXISTS (
  SELECT 1 FROM workspaces w
  WHERE w.id = sqlc.arg(workspace_id) AND w.enabled = TRUE AND w.deleted_at_m IS NULL
    AND (sqlc.arg(project_id) = '' OR EXISTS (SELECT 1 FROM projects p WHERE p.id = sqlc.arg(project_id) AND p.workspace_id = w.id))
    AND (sqlc.arg(app_id) = '' OR EXISTS (SELECT 1 FROM apps a WHERE a.id = sqlc.arg(app_id) AND a.project_id = sqlc.arg(project_id) AND a.workspace_id = w.id))
    AND (sqlc.arg(environment_id) = '' OR EXISTS (SELECT 1 FROM environments e WHERE e.id = sqlc.arg(environment_id) AND e.app_id = sqlc.arg(app_id) AND e.workspace_id = w.id))
) AS valid;

-- name: NotificationEmailEnabled :one
SELECT NOT EXISTS (
  SELECT 1 FROM notification_preferences
  WHERE workspace_id = sqlc.arg(workspace_id) AND user_id = sqlc.arg(user_id)
    AND notification_type = sqlc.arg(notification_type) AND channel = 'email' AND enabled = FALSE
) AS enabled;

-- name: NotificationDestinationEnabled :one
SELECT EXISTS (
  SELECT 1 FROM notification_destinations
  WHERE workspace_id = sqlc.arg(workspace_id) AND id = sqlc.arg(id) AND channel = sqlc.arg(channel) AND enabled = TRUE
) AS enabled;
