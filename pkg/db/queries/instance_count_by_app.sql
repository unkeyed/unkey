-- name: CountInstancesByAppId :one
SELECT COUNT(*) as count
FROM instances i
JOIN deployments d ON i.deployment_id = d.id
WHERE d.app_id = sqlc.arg(app_id);
