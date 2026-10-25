-- name: InsertNotification :exec
INSERT INTO notifications (id, workspace_id, source_event_id, notification_type, request_hash, schema_version, resource_context, payload, occurred_at, expires_at, created_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(source_event_id), sqlc.arg(notification_type), sqlc.arg(request_hash), sqlc.arg(schema_version), sqlc.arg(resource_context), sqlc.arg(payload), sqlc.arg(occurred_at), sqlc.narg(expires_at), sqlc.arg(created_at));
