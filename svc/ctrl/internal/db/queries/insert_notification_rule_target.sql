-- name: InsertNotificationRuleTarget :exec
INSERT INTO notification_subscription_targets (id, workspace_id, subscription_id, channel, audience, user_id, destination_id, created_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(subscription_id), sqlc.arg(channel), sqlc.narg(audience), sqlc.narg(user_id), sqlc.narg(destination_id), sqlc.arg(created_at));
