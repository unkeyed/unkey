-- name: DeleteHorizontalAutoscalingPolicy :exec
DELETE FROM horizontal_autoscaling_policies
WHERE id = sqlc.arg(id)
  AND workspace_id = sqlc.arg(workspace_id);
