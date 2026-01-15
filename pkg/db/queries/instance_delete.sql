

-- name: DeleteInstance :exec
DELETE FROM instances WHERE k8s_name = sqlc.arg(k8s_name) AND region = sqlc.arg(region);
