-- name: DeleteConnectionAppTargetByConnectionId :exec
DELETE FROM connection_app_targets WHERE connection_id = sqlc.arg(connection_id);
