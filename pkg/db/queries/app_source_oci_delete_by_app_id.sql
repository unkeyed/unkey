-- name: DeleteAppSourceOciByAppId :exec
DELETE FROM app_source_oci
WHERE app_id = sqlc.arg(app_id);
