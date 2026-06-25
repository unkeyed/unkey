-- name: UpdateHorizontalAutoscalingPolicy :exec
UPDATE horizontal_autoscaling_policies
SET
    replicas_min = sqlc.arg(replicas_min),
    replicas_max = sqlc.arg(replicas_max),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
  AND workspace_id = sqlc.arg(workspace_id);
